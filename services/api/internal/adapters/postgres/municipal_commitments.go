package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const municipalCommitmentsShown = 50

type MunicipalCommitmentRepo struct{ db *sql.DB }

func NewMunicipalCommitmentRepo(db *sql.DB) *MunicipalCommitmentRepo {
	return &MunicipalCommitmentRepo{db: db}
}

func (r *MunicipalCommitmentRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM municipal_commitments, municipal_totals LIMIT 0`)
	return err
}

func (r *MunicipalCommitmentRepo) ReplaceMunicipalYear(ctx context.Context, year int, commitments []domain.MunicipalCommitment, totals []domain.MunicipalTotal) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, table := range []string{"municipal_commitments", "municipal_totals"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE year = $1`, year); err != nil {
			return err
		}
	}
	if err := copyMunicipalCommitments(ctx, tx, commitments); err != nil {
		return err
	}
	for _, t := range totals {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO municipal_totals (year, entity_id, entity, committed_cents, liquidated_cents, paid_cents) VALUES ($1, $2, $3, $4, $5, $6)`,
			t.Year, t.EntityID, t.Entity, t.CommittedCents, t.LiquidatedCents, t.PaidCents); err != nil {
			return err
		}
	}
	if err := relinkMunicipalYear(ctx, tx, year); err != nil {
		return err
	}
	return tx.Commit()
}

func copyMunicipalCommitments(ctx context.Context, tx *sql.Tx, commitments []domain.MunicipalCommitment) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("municipal_commitments", "entity_id", "entity", "year", "commitment_id", "number", "committed_on",
		"cnpj", "name", "object", "process_kind", "process", "modality", "committed_cents", "liquidated_cents", "paid_cents"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for _, c := range commitments {
		if _, err := stmt.ExecContext(ctx, c.EntityID, c.Entity, c.Year, c.CommitmentID, c.Number, c.Date, c.CNPJ, c.Name, c.Object, c.ProcessKind,
			c.Process, c.Modality, c.CommittedCents, c.LiquidatedCents, c.PaidCents); err != nil {
			return fmt.Errorf("empenho %d de %d: %w", c.CommitmentID, c.Year, err)
		}
	}
	return closeCopy(ctx, stmt)
}

func relinkMunicipalYear(ctx context.Context, tx *sql.Tx, year int) error {
	source, record := domain.SourceMunicipalCommitments, strconv.Itoa(year)
	if _, err := tx.ExecContext(ctx, `DELETE FROM entity_links WHERE source = $1 AND record_kind = $2 AND record_id = $3`,
		source, domain.RecordPaymentYear, record); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO entity_links (entity_id, source, record_kind, record_id, role, certainty, evidence)
		SELECT e.id, $1, $2, $3, $4, $5, 'empenhos do portal de ' || $3
		FROM municipal_commitments m JOIN entities e ON e.kind = 'cnpj' AND e.key = m.cnpj
		WHERE m.year = $6
		GROUP BY e.id
		ON CONFLICT DO NOTHING`,
		source, domain.RecordPaymentYear, record, paymentLinkRole, domain.CertaintyExact, year)
	return err
}

func (r *MunicipalCommitmentRepo) MunicipalByCNPJ(ctx context.Context, cnpj string) (domain.MunicipalSupplier, error) {
	var out domain.MunicipalSupplier
	years, err := r.db.QueryContext(ctx, `
		SELECT year, count(*), sum(committed_cents), sum(paid_cents) FROM municipal_commitments WHERE cnpj = $1 GROUP BY year ORDER BY year DESC`, cnpj)
	if err != nil {
		return out, err
	}
	defer years.Close()
	for years.Next() {
		var y domain.MunicipalYear
		if err := years.Scan(&y.Year, &y.Commitments, &y.CommittedCents, &y.PaidCents); err != nil {
			return out, err
		}
		out.Years = append(out.Years, y)
	}
	if err := years.Err(); err != nil {
		return out, err
	}
	out.Recent, err = r.recent(ctx, cnpj)
	return out, err
}

func (r *MunicipalCommitmentRepo) recent(ctx context.Context, cnpj string) ([]domain.MunicipalCommitment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT entity_id, entity, year, commitment_id, number, committed_on, cnpj, name, object, process_kind, process, modality,
			committed_cents, liquidated_cents, paid_cents
		FROM municipal_commitments WHERE cnpj = $1 ORDER BY committed_on DESC, commitment_id DESC LIMIT $2`, cnpj, municipalCommitmentsShown)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MunicipalCommitment
	for rows.Next() {
		var c domain.MunicipalCommitment
		if err := rows.Scan(&c.EntityID, &c.Entity, &c.Year, &c.CommitmentID, &c.Number, &c.Date, &c.CNPJ, &c.Name, &c.Object, &c.ProcessKind,
			&c.Process, &c.Modality, &c.CommittedCents, &c.LiquidatedCents, &c.PaidCents); err != nil {
			return nil, err
		}
		c.Date = c.Date.UTC()
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *MunicipalCommitmentRepo) MunicipalPaidByYear(ctx context.Context) (map[int]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT year, sum(paid_cents) FROM municipal_totals GROUP BY year`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]int64{}
	for rows.Next() {
		var year int
		var paid int64
		if err := rows.Scan(&year, &paid); err != nil {
			return nil, err
		}
		out[year] = paid
	}
	return out, rows.Err()
}
