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
		INSERT INTO gazettes (edition_number, published_at, source_url, storage_path, checksum, indexed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (checksum) DO NOTHING
		RETURNING id`,
		g.EditionNumber, g.PublishedAt.Format("2006-01-02"), g.SourceURL, g.StoragePath, g.Checksum, g.IndexedAt,
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
		SELECT id, edition_number, published_at, is_extra, source_url, storage_path, checksum, indexed_at
		FROM gazettes WHERE id = $1`, id,
	).Scan(&g.ID, &g.EditionNumber, &g.PublishedAt, &g.IsExtra, &g.SourceURL, &g.StoragePath, &g.Checksum, &g.IndexedAt)
	return g, notFound(err)
}

func (r *GazetteRepo) ListByPeriod(ctx context.Context, from, to time.Time) ([]domain.Gazette, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, edition_number, published_at, is_extra, source_url, storage_path, checksum, indexed_at
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
		if err := rows.Scan(&g.ID, &g.EditionNumber, &g.PublishedAt, &g.IsExtra, &g.SourceURL, &g.StoragePath, &g.Checksum, &g.IndexedAt); err != nil {
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

	if _, err := tx.ExecContext(ctx, `DELETE FROM acts WHERE gazette_id = $1`, gazetteID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE gazettes SET edition_number = $2 WHERE id = $1`, gazetteID, editionNumber); err != nil {
		return err
	}
	if err := insertActs(ctx, tx, gazetteID, acts); err != nil {
		return err
	}
	return tx.Commit()
}
