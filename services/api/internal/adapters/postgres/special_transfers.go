package postgres

import (
	"context"
	"database/sql"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type SpecialTransferRepo struct{ db *sql.DB }

func NewSpecialTransferRepo(db *sql.DB) *SpecialTransferRepo { return &SpecialTransferRepo{db: db} }

func (r *SpecialTransferRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM special_transfers, special_transfer_executors LIMIT 0`)
	return err
}

func (r *SpecialTransferRepo) ReplaceSpecialTransfers(ctx context.Context, transfers []domain.SpecialTransfer) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM special_transfers`); err != nil {
		return err
	}
	for _, st := range transfers {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO special_transfers (plan_id, code, year, status, author, amendment, area, value_cents, committed_cents, paid_cents,
				last_paid_at, work_plan_status, execution_end, report_kind, report_at, executed_cents, pending_cents)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
			st.PlanID, st.Code, st.Year, st.Status, st.Author, st.Amendment, st.Area, st.ValueCents, st.CommittedCents, st.PaidCents,
			st.LastPaidAt, st.WorkPlanStatus, st.ExecutionEnd, st.ReportKind, st.ReportAt, st.ExecutedCents, st.PendingCents); err != nil {
			return err
		}
		for i, e := range st.Executors {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO special_transfer_executors (plan_id, position, cnpj, name, object, value_cents) VALUES ($1, $2, $3, $4, $5, $6)`,
				st.PlanID, i, e.CNPJ, e.Name, e.Object, e.ValueCents); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (r *SpecialTransferRepo) SpecialTransfers(ctx context.Context) ([]domain.SpecialTransfer, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT plan_id, code, year, status, author, amendment, area, value_cents, committed_cents, paid_cents, last_paid_at,
			work_plan_status, execution_end, report_kind, report_at, executed_cents, pending_cents
		FROM special_transfers ORDER BY year DESC, value_cents DESC, plan_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SpecialTransfer
	index := map[int64]int{}
	for rows.Next() {
		var st domain.SpecialTransfer
		if err := rows.Scan(&st.PlanID, &st.Code, &st.Year, &st.Status, &st.Author, &st.Amendment, &st.Area, &st.ValueCents, &st.CommittedCents,
			&st.PaidCents, &st.LastPaidAt, &st.WorkPlanStatus, &st.ExecutionEnd, &st.ReportKind, &st.ReportAt, &st.ExecutedCents, &st.PendingCents); err != nil {
			return nil, err
		}
		index[st.PlanID] = len(out)
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, r.addExecutors(ctx, out, index)
}

func (r *SpecialTransferRepo) addExecutors(ctx context.Context, out []domain.SpecialTransfer, index map[int64]int) error {
	rows, err := r.db.QueryContext(ctx, `SELECT plan_id, cnpj, name, object, value_cents FROM special_transfer_executors ORDER BY plan_id, position`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var planID int64
		var e domain.SpecialExecutor
		if err := rows.Scan(&planID, &e.CNPJ, &e.Name, &e.Object, &e.ValueCents); err != nil {
			return err
		}
		if i, ok := index[planID]; ok {
			out[i].Executors = append(out[i].Executors, e)
		}
	}
	return rows.Err()
}
