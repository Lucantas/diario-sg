package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const paymentLinkRole = "credor"

type PaymentRepo struct{ db *sql.DB }

func NewPaymentRepo(db *sql.DB) *PaymentRepo { return &PaymentRepo{db: db} }

func (r *PaymentRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM payments LIMIT 0`)
	return err
}

func (r *PaymentRepo) ReplaceYear(ctx context.Context, source string, year int, payments []domain.Payment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM payments WHERE source = $1 AND year = $2`, source, year); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("payments", "source", "year", "month", "unit", "commitment", "cnpj", "function",
		"committed_cents", "liquidated_cents", "paid_cents"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for _, p := range payments {
		if _, err := stmt.ExecContext(ctx, p.Source, p.Year, p.Month, p.Unit, p.Commitment, p.CNPJ, p.Function,
			p.CommittedCents, p.LiquidatedCents, p.PaidCents); err != nil {
			return fmt.Errorf("empenho %s de %d: %w", p.Commitment, p.Year, err)
		}
	}
	if err := closeCopy(ctx, stmt); err != nil {
		return err
	}
	if err := relinkPaymentYear(ctx, tx, source, year); err != nil {
		return err
	}
	return tx.Commit()
}

func relinkPaymentYear(ctx context.Context, tx *sql.Tx, source string, year int) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM entity_links WHERE source = $1 AND record_kind = $2 AND record_id = $3`,
		source, domain.RecordPaymentYear, strconv.Itoa(year)); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO entity_links (entity_id, source, record_kind, record_id, role, certainty, evidence)
		SELECT e.id, $1, $2, $3, $4, $5, 'empenhos de ' || $3
		FROM payments p JOIN entities e ON e.kind = 'cnpj' AND e.key = p.cnpj
		WHERE p.source = $1 AND p.year = $6
		GROUP BY e.id
		ON CONFLICT DO NOTHING`,
		source, domain.RecordPaymentYear, strconv.Itoa(year), paymentLinkRole, domain.CertaintyExact, year)
	return err
}

func (r *PaymentRepo) PaymentsByCNPJ(ctx context.Context, cnpj string) ([]domain.PaymentYear, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT year, array_agg(DISTINCT unit ORDER BY unit), sum(committed_cents), sum(liquidated_cents), sum(paid_cents)
		FROM payments WHERE cnpj = $1 GROUP BY year ORDER BY year DESC`, cnpj)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.PaymentYear{}
	for rows.Next() {
		var y domain.PaymentYear
		if err := rows.Scan(&y.Year, pq.Array(&y.Units), &y.CommittedCents, &y.LiquidatedCents, &y.PaidCents); err != nil {
			return nil, err
		}
		out = append(out, y)
	}
	return out, rows.Err()
}

func (r *PaymentRepo) PaymentsCoverage(ctx context.Context) (*domain.PaymentCoverage, error) {
	var from, to sql.NullInt64
	if err := r.db.QueryRowContext(ctx, `SELECT min(year * 100 + month), max(year * 100 + month) FROM payments WHERE paid_cents <> 0`).Scan(&from, &to); err != nil {
		return nil, err
	}
	if !from.Valid {
		return nil, nil
	}
	return &domain.PaymentCoverage{FromYear: int(from.Int64 / 100), FromMonth: int(from.Int64 % 100),
		ToYear: int(to.Int64 / 100), ToMonth: int(to.Int64 % 100)}, nil
}

func (r *PaymentRepo) PaidByCNPJYear(ctx context.Context) ([]domain.PaidTotal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT cnpj, year, unit ILIKE 'C_MARA%', sum(paid_cents) FROM payments GROUP BY 1, 2, 3`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PaidTotal
	for rows.Next() {
		var t domain.PaidTotal
		if err := rows.Scan(&t.CNPJ, &t.Year, &t.Chamber, &t.PaidCents); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
