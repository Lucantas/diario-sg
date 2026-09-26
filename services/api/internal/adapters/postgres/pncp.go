package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const pncpLinkRole = "contratado"

type PNCPRepo struct{ db *sql.DB }

func NewPNCPRepo(db *sql.DB) *PNCPRepo { return &PNCPRepo{db: db} }

func (r *PNCPRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM pncp_contracts LIMIT 0`)
	return err
}

func (r *PNCPRepo) ReplaceYears(ctx context.Context, from, to int, contracts []domain.PNCPContract) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM pncp_contracts WHERE year BETWEEN $1 AND $2`, from, to); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO pncp_contracts (control_number, org_cnpj, unit_name, year, sequence, kind, process, number, supplier_cnpj,
			supplier_name, object, value_cents, signed_at, published_at, starts_at, ends_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (control_number) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, c := range contracts {
		if _, err := stmt.ExecContext(ctx, c.ControlNumber, c.OrgCNPJ, c.UnitName, c.Year, c.Sequence, c.Kind, c.Process, c.Number,
			c.SupplierCNPJ, c.SupplierName, c.Object, c.ValueCents, c.SignedAt, c.PublishedAt, c.StartsAt, c.EndsAt); err != nil {
			return fmt.Errorf("contrato %s: %w", c.ControlNumber, err)
		}
	}
	if err := relinkPNCP(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func relinkPNCP(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM entity_links WHERE source = $1`, domain.SourcePNCP); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO entity_links (entity_id, source, record_kind, record_id, role, certainty, evidence)
		SELECT e.id, $1, $2, c.control_number, $3, $4, c.number || ' · ' || c.process
		FROM pncp_contracts c JOIN entities e ON e.kind = 'cnpj' AND e.key = c.supplier_cnpj
		ON CONFLICT DO NOTHING`, domain.SourcePNCP, domain.RecordPNCP, pncpLinkRole, domain.CertaintyExact)
	return err
}

const pncpColumns = `control_number, org_cnpj, unit_name, year, sequence, kind, process, number, supplier_cnpj, supplier_name,
	object, value_cents, signed_at, published_at, starts_at, ends_at`

func scanPNCP(rows *sql.Rows) ([]domain.PNCPContract, error) {
	out := []domain.PNCPContract{}
	for rows.Next() {
		var c domain.PNCPContract
		if err := rows.Scan(&c.ControlNumber, &c.OrgCNPJ, &c.UnitName, &c.Year, &c.Sequence, &c.Kind, &c.Process, &c.Number,
			&c.SupplierCNPJ, &c.SupplierName, &c.Object, &c.ValueCents, &c.SignedAt, &c.PublishedAt, &c.StartsAt, &c.EndsAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PNCPRepo) PNCPContractsBySupplier(ctx context.Context, cnpj string) ([]domain.PNCPContract, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+pncpColumns+` FROM pncp_contracts WHERE supplier_cnpj = $1
		ORDER BY signed_at DESC NULLS LAST, control_number`, cnpj)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPNCP(rows)
}

func (r *PNCPRepo) AllPNCPContracts(ctx context.Context) ([]domain.PNCPContract, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+pncpColumns+` FROM pncp_contracts ORDER BY signed_at, control_number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPNCP(rows)
}
