package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const amendmentPaymentLinkRole = "favorecido"

type FederalRepo struct{ db *sql.DB }

func NewFederalRepo(db *sql.DB) *FederalRepo { return &FederalRepo{db: db} }

func (r *FederalRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM federal_amendments, federal_amendment_payments, federal_transfers LIMIT 0`)
	return err
}

func (r *FederalRepo) ReplaceAmendments(ctx context.Context, amendments []domain.Amendment, payments []domain.AmendmentPayment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `TRUNCATE federal_amendments, federal_amendment_payments`); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("federal_amendments", "code", "year", "kind", "author", "number", "function", "subfunction",
		"program", "action", "committed_cents", "liquidated_cents", "paid_cents"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for _, a := range amendments {
		if _, err := stmt.ExecContext(ctx, a.Code, a.Year, a.Kind, a.Author, a.Number, a.Function, a.Subfunction, a.Program, a.Action,
			a.CommittedCents, a.LiquidatedCents, a.PaidCents); err != nil {
			return err
		}
	}
	if err := closeCopy(ctx, stmt); err != nil {
		return err
	}
	stmt, err = tx.PrepareContext(ctx, pq.CopyIn("federal_amendment_payments", "code", "author", "kind", "month", "cnpj", "name",
		"legal_nature", "value_cents"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for _, p := range payments {
		if _, err := stmt.ExecContext(ctx, p.Code, p.Author, p.Kind, p.Month, p.CNPJ, p.Name, p.LegalNature, p.ValueCents); err != nil {
			return err
		}
	}
	if err := closeCopy(ctx, stmt); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM entity_links WHERE source = $1`, domain.SourceAmendments); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO entity_links (entity_id, source, record_kind, record_id, role, certainty, evidence)
		SELECT e.id, $1, $2, p.id::text, $3, $4, 'emenda ' || p.code
		FROM federal_amendment_payments p JOIN entities e ON e.kind = 'cnpj' AND e.key = p.cnpj
		ON CONFLICT DO NOTHING`, domain.SourceAmendments, domain.RecordAmendmentPayment, amendmentPaymentLinkRole, domain.CertaintyExact); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *FederalRepo) ReplaceTransferMonth(ctx context.Context, month time.Time, transfers []domain.FederalTransfer) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM federal_transfers WHERE month = $1`, month); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("federal_transfers", "month", "kind", "organ", "function", "program", "action", "label",
		"cnpj", "name", "value_cents"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for _, t := range transfers {
		if _, err := stmt.ExecContext(ctx, month, t.Kind, t.Organ, t.Function, t.Program, t.Action, t.Label, t.CNPJ, t.Name, t.ValueCents); err != nil {
			return err
		}
	}
	if err := closeCopy(ctx, stmt); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *FederalRepo) Amendments(ctx context.Context) ([]domain.Amendment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT code, year, kind, author, number, function, subfunction, program, action,
		committed_cents, liquidated_cents, paid_cents FROM federal_amendments ORDER BY year DESC, paid_cents DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Amendment{}
	for rows.Next() {
		var a domain.Amendment
		if err := rows.Scan(&a.Code, &a.Year, &a.Kind, &a.Author, &a.Number, &a.Function, &a.Subfunction, &a.Program, &a.Action,
			&a.CommittedCents, &a.LiquidatedCents, &a.PaidCents); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *FederalRepo) payments(ctx context.Context, where string, args ...any) ([]domain.AmendmentPayment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT code, author, kind, month, cnpj, name, legal_nature, value_cents
		FROM federal_amendment_payments `+where+` ORDER BY month DESC, value_cents DESC, id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AmendmentPayment{}
	for rows.Next() {
		var p domain.AmendmentPayment
		if err := rows.Scan(&p.Code, &p.Author, &p.Kind, &p.Month, &p.CNPJ, &p.Name, &p.LegalNature, &p.ValueCents); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *FederalRepo) AmendmentPayments(ctx context.Context) ([]domain.AmendmentPayment, error) {
	return r.payments(ctx, "")
}

func (r *FederalRepo) AmendmentPaymentsByCNPJ(ctx context.Context, cnpj string) ([]domain.AmendmentPayment, error) {
	return r.payments(ctx, "WHERE cnpj = $1", cnpj)
}

func (r *FederalRepo) TransferMonths(ctx context.Context) (*time.Time, *time.Time, error) {
	var from, to sql.NullTime
	if err := r.db.QueryRowContext(ctx, `SELECT min(month), max(month) FROM federal_transfers`).Scan(&from, &to); err != nil || !from.Valid {
		return nil, nil, err
	}
	return &from.Time, &to.Time, nil
}

func (r *FederalRepo) TransferTotals(ctx context.Context) ([]domain.TransferTotal, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT extract(year FROM month)::int, kind, function, sum(value_cents)
		FROM federal_transfers GROUP BY 1, 2, 3 ORDER BY 1 DESC, 4 DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TransferTotal{}
	for rows.Next() {
		var t domain.TransferTotal
		if err := rows.Scan(&t.Year, &t.Kind, &t.Function, &t.ValueCents); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
