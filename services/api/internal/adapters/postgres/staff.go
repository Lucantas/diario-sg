package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type StaffRepo struct{ db *sql.DB }

func NewStaffRepo(db *sql.DB) *StaffRepo { return &StaffRepo{db: db} }

func (r *StaffRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM tce_staff LIMIT 0`)
	return err
}

func (r *StaffRepo) ReplaceStaffYear(ctx context.Context, year int, rows []domain.StaffRow) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM tce_staff WHERE extract(year FROM month) = $1`, year); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("tce_staff", "month", "unit", "situation", "grp", "headcount", "remuneration_cents"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for _, s := range rows {
		if _, err := stmt.ExecContext(ctx, s.Month, s.Unit, s.Situation, s.Group, s.Headcount, s.RemunerationCents); err != nil {
			return fmt.Errorf("pessoal de %s: %w", s.Month.Format("01/2006"), err)
		}
	}
	if err := closeCopy(ctx, stmt); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *StaffRepo) StaffRows(ctx context.Context, unit string) ([]domain.StaffRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT month, unit, situation, grp, headcount, remuneration_cents FROM tce_staff
		WHERE $1 = '' OR unit = $1 ORDER BY month, unit, situation`, unit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.StaffRow
	for rows.Next() {
		var s domain.StaffRow
		if err := rows.Scan(&s.Month, &s.Unit, &s.Situation, &s.Group, &s.Headcount, &s.RemunerationCents); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *StaffRepo) StaffUnits(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT unit FROM tce_staff ORDER BY unit`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
