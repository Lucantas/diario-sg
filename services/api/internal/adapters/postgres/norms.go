package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const normColumns = `kind, number, year, suffix, author, summary, promulgated_on, text_url`

type NormRepo struct{ db *sql.DB }

func NewNormRepo(db *sql.DB) *NormRepo { return &NormRepo{db: db} }

func (r *NormRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM norms LIMIT 0`)
	return err
}

func (r *NormRepo) ReplaceNorms(ctx context.Context, norms []domain.Norm) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM norms`); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("norms", "kind", "number", "year", "suffix", "author", "summary", "promulgated_on", "text_url"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for _, n := range norms {
		if _, err := stmt.ExecContext(ctx, string(n.Kind), n.Number, n.Year, n.Suffix, n.Author, n.Summary, n.PromulgatedOn, n.TextURL); err != nil {
			return fmt.Errorf("norma %s %s: %w", n.Kind, n.Label(), err)
		}
	}
	if err := closeCopy(ctx, stmt); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *NormRepo) NormsByNumber(ctx context.Context, kind domain.NormKind, number, year int) ([]domain.Norm, error) {
	return r.query(ctx, `SELECT `+normColumns+` FROM norms WHERE kind = $1 AND number = $2 AND year = $3 ORDER BY suffix`, string(kind), number, year)
}

func (r *NormRepo) SearchNorms(ctx context.Context, kind domain.NormKind, text string, limit int) ([]domain.Norm, error) {
	return r.query(ctx, `SELECT `+normColumns+` FROM norms
		WHERE ($1 = '' OR kind = $1) AND to_tsvector('portuguese_unaccent', summary || ' ' || author) @@ plainto_tsquery('portuguese_unaccent', $2)
		ORDER BY promulgated_on DESC NULLS LAST, year DESC, number DESC LIMIT $3`, string(kind), text, limit)
}

func (r *NormRepo) query(ctx context.Context, q string, args ...any) ([]domain.Norm, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Norm{}
	for rows.Next() {
		var n domain.Norm
		var kind string
		if err := rows.Scan(&kind, &n.Number, &n.Year, &n.Suffix, &n.Author, &n.Summary, &n.PromulgatedOn, &n.TextURL); err != nil {
			return nil, err
		}
		n.Kind = domain.NormKind(kind)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *NormRepo) NormsCitingBills(ctx context.Context) ([]domain.Norm, error) {
	return r.query(ctx, `SELECT `+normColumns+` FROM norms WHERE unaccent(author) ~* 'projeto d' ORDER BY year, number`)
}
