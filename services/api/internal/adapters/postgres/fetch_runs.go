package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type FetchRunRepo struct{ db *sql.DB }

func NewFetchRunRepo(db *sql.DB) *FetchRunRepo { return &FetchRunRepo{db: db} }

func (r *FetchRunRepo) Save(ctx context.Context, run domain.FetchRun) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fetch_runs (id, source, requested_from, requested_to, found, stored, skipped, failed, error, started_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO NOTHING`,
		run.ID, run.Source, run.RequestedFrom.Format(time.DateOnly), run.RequestedTo.Format(time.DateOnly),
		run.Found, run.Stored, run.Skipped, run.Failed, run.Error, run.StartedAt, run.FinishedAt)
	return notFound(err)
}
