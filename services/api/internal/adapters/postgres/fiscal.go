package postgres

import (
	"context"
	"database/sql"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type FiscalRepo struct{ db *sql.DB }

func NewFiscalRepo(db *sql.DB) *FiscalRepo { return &FiscalRepo{db: db} }

func (r *FiscalRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM fiscal_totals LIMIT 0`)
	return err
}

func (r *FiscalRepo) SaveFiscalTotals(ctx context.Context, totals []domain.FiscalTotal) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, t := range totals {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO fiscal_totals (year, period, committed_cents, liquidated_cents, paid_cents, source_url) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (year) DO UPDATE SET period = excluded.period, committed_cents = excluded.committed_cents,
				liquidated_cents = excluded.liquidated_cents, paid_cents = excluded.paid_cents, source_url = excluded.source_url`,
			t.Year, t.Period, t.CommittedCents, t.LiquidatedCents, t.PaidCents, t.SourceURL); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *FiscalRepo) FiscalTotals(ctx context.Context) ([]domain.FiscalTotal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT year, period, committed_cents, liquidated_cents, paid_cents, source_url FROM fiscal_totals ORDER BY year`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.FiscalTotal
	for rows.Next() {
		var t domain.FiscalTotal
		if err := rows.Scan(&t.Year, &t.Period, &t.CommittedCents, &t.LiquidatedCents, &t.PaidCents, &t.SourceURL); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *FiscalRepo) PaidByYear(ctx context.Context) ([]domain.YearPaid, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT year, coalesce(sum(committed_cents), 0), coalesce(sum(paid_cents), 0) FROM payments GROUP BY year ORDER BY year`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.YearPaid
	for rows.Next() {
		var y domain.YearPaid
		if err := rows.Scan(&y.Year, &y.CommittedCents, &y.PaidCents); err != nil {
			return nil, err
		}
		out = append(out, y)
	}
	return out, rows.Err()
}
