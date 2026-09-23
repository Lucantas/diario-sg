package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type GazetteRepo struct{ db *sql.DB }

func NewGazetteRepo(db *sql.DB) *GazetteRepo { return &GazetteRepo{db: db} }

func (r *GazetteRepo) FindIDByChecksum(ctx context.Context, checksum string) (string, bool, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM gazettes WHERE checksum = $1`, checksum).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return id, err == nil, err
}

func (r *GazetteRepo) SaveWithActs(ctx context.Context, g *domain.Gazette, acts []domain.Act) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	err = tx.QueryRowContext(ctx, `
		INSERT INTO gazettes (edition_number, published_at, source_url, storage_path, checksum, indexed_at, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (checksum) DO NOTHING
		RETURNING id`,
		g.EditionNumber, g.PublishedAt.Format("2006-01-02"), g.SourceURL, g.StoragePath, g.Checksum, g.IndexedAt, domain.SourceOrDefault(g.Source),
	).Scan(&g.ID)
	if errors.Is(err, sql.ErrNoRows) {

		id, _, ferr := r.FindIDByChecksum(ctx, g.Checksum)
		g.ID = id
		return ferr
	}
	if err != nil {
		return err
	}

	if err := insertActs(ctx, tx, g.ID, acts); err != nil {
		return err
	}
	if err := linkActs(ctx, tx, g.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func insertActs(ctx context.Context, tx *sql.Tx, gazetteID string, acts []domain.Act) error {
	insertAct, err := tx.PrepareContext(ctx, `
		INSERT INTO acts (gazette_id, type, title, body, position, page_start, page_end, organ)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`)
	if err != nil {
		return err
	}
	defer insertAct.Close()
	insertEntity, err := tx.PrepareContext(ctx, `
		INSERT INTO act_entities (act_id, kind, value, normalized) VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING`)
	if err != nil {
		return err
	}
	defer insertEntity.Close()
	for _, a := range acts {
		var actID string
		if err := insertAct.QueryRowContext(ctx, gazetteID, string(a.Type), a.Title, a.Body, a.Position,
			a.PageStart, a.PageEnd, a.Organ).Scan(&actID); err != nil {
			return fmt.Errorf("ato %d: %w", a.Position, err)
		}
		for _, e := range a.Entities {
			if _, err := insertEntity.ExecContext(ctx, actID, string(e.Kind), e.Value, e.Normalized); err != nil {
				return fmt.Errorf("ato %d, entidade %s: %w", a.Position, e.Kind, err)
			}
		}
	}
	return nil
}

func (r *GazetteRepo) FindByID(ctx context.Context, id string) (domain.Gazette, error) {
	var g domain.Gazette
	err := r.db.QueryRowContext(ctx, `
		SELECT id, edition_number, published_at, is_extra, source_url, storage_path, checksum, indexed_at, source
		FROM gazettes WHERE id = $1`, id,
	).Scan(&g.ID, &g.EditionNumber, &g.PublishedAt, &g.IsExtra, &g.SourceURL, &g.StoragePath, &g.Checksum, &g.IndexedAt, &g.Source)
	return g, notFound(err)
}

func (r *GazetteRepo) ListByPeriod(ctx context.Context, from, to time.Time) ([]domain.Gazette, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, edition_number, published_at, is_extra, source_url, storage_path, checksum, indexed_at, source
		FROM gazettes
		WHERE published_at BETWEEN $1::date AND $2::date
		ORDER BY published_at, source_url`, from.Format(time.DateOnly), to.Format(time.DateOnly))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Gazette
	for rows.Next() {
		var g domain.Gazette
		if err := rows.Scan(&g.ID, &g.EditionNumber, &g.PublishedAt, &g.IsExtra, &g.SourceURL, &g.StoragePath, &g.Checksum, &g.IndexedAt, &g.Source); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *GazetteRepo) ReplaceActs(ctx context.Context, gazetteID, editionNumber string, acts []domain.Act) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := unlinkActs(ctx, tx, gazetteID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM acts WHERE gazette_id = $1`, gazetteID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE gazettes SET edition_number = $2 WHERE id = $1`, gazetteID, editionNumber); err != nil {
		return err
	}
	if err := insertActs(ctx, tx, gazetteID, acts); err != nil {
		return err
	}
	if err := linkActs(ctx, tx, gazetteID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *GazetteRepo) Coverage(ctx context.Context) (domain.Coverage, error) {
	var c domain.Coverage
	err := r.db.QueryRowContext(ctx, `
		SELECT coalesce(min(published_at), 'epoch'), coalesce(max(published_at), 'epoch'),
		       coalesce(max(indexed_at), 'epoch'), count(*), (SELECT count(*) FROM acts)
		FROM gazettes`,
	).Scan(&c.First, &c.Last, &c.LastIndexedAt, &c.Gazettes, &c.Acts)
	if err != nil {
		return c, err
	}
	run := &c.LastRun
	err = r.db.QueryRowContext(ctx, `
		SELECT id, source, requested_from, requested_to, found, stored, skipped, failed, error, started_at, finished_at
		FROM fetch_runs WHERE source = $1
		ORDER BY finished_at DESC LIMIT 1`, domain.SourceDiarioPrefeitura,
	).Scan(&run.ID, &run.Source, &run.RequestedFrom, &run.RequestedTo, &run.Found, &run.Stored, &run.Skipped, &run.Failed, &run.Error, &run.StartedAt, &run.FinishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return c, nil
	}
	return c, err
}
