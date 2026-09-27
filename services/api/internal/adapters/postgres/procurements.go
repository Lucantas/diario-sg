package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type ProcurementRepo struct{ db *sql.DB }

func NewProcurementRepo(db *sql.DB) *ProcurementRepo { return &ProcurementRepo{db: db} }

func (r *ProcurementRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM procurements, procurement_contracts LIMIT 0`)
	return err
}

func (r *ProcurementRepo) ReplaceMural(ctx context.Context, procurements []domain.Procurement, contracts []domain.ProcurementContract) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM procurements; DELETE FROM procurement_contracts`); err != nil {
		return err
	}
	if err := copyProcurements(ctx, tx, procurements); err != nil {
		return err
	}
	if err := copyProcurementContracts(ctx, tx, contracts); err != nil {
		return err
	}
	return tx.Commit()
}

func copyProcurements(ctx context.Context, tx *sql.Tx, procurements []domain.Procurement) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("procurements", "list", "id", "notice", "process", "process_key", "modality", "criterion",
		"opens_at", "object", "status", "url"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for _, p := range procurements {
		if _, err := stmt.ExecContext(ctx, p.List, p.ID, p.Notice, p.Process, p.ProcessKey, p.Modality, p.Criterion, p.OpensAt, p.Object, p.Status, p.URL); err != nil {
			return fmt.Errorf("licitação %d: %w", p.ID, err)
		}
	}
	return closeCopy(ctx, stmt)
}

func copyProcurementContracts(ctx context.Context, tx *sql.Tx, contracts []domain.ProcurementContract) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("procurement_contracts", "position", "procurement_id", "notice", "process", "process_key",
		"modality", "object", "value_cents", "supplier", "instrument", "document_url"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for i, c := range contracts {
		if _, err := stmt.ExecContext(ctx, i, c.ProcurementID, c.Notice, c.Process, c.ProcessKey, c.Modality, c.Object, c.ValueCents, c.Supplier,
			c.Instrument, c.DocumentURL); err != nil {
			return fmt.Errorf("contrato da licitação %d: %w", c.ProcurementID, err)
		}
	}
	return closeCopy(ctx, stmt)
}

func (r *ProcurementRepo) MuralByCNPJ(ctx context.Context, cnpj string) (domain.MuralMatches, error) {
	var out domain.MuralMatches
	keys, err := r.commitmentProcessKeys(ctx, cnpj)
	if err != nil || len(keys) == 0 {
		return out, err
	}
	if out.Procurements, err = r.procurementsByKeys(ctx, keys); err != nil {
		return out, err
	}
	out.Contracts, err = r.contractsByKeys(ctx, keys)
	return out, err
}

func (r *ProcurementRepo) commitmentProcessKeys(ctx context.Context, cnpj string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT process FROM municipal_commitments WHERE cnpj = $1`, cnpj)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var process string
		if err := rows.Scan(&process); err != nil {
			return nil, err
		}
		if key := domain.ProcessKey(process); key != "" {
			keys = append(keys, key)
		}
	}
	return keys, rows.Err()
}

func (r *ProcurementRepo) procurementsByKeys(ctx context.Context, keys []string) ([]domain.Procurement, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT list, id, notice, process, process_key, modality, criterion, opens_at, object, status, url
		FROM procurements WHERE process_key = ANY($1) ORDER BY opens_at DESC NULLS LAST, id DESC`, pq.Array(keys))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Procurement
	for rows.Next() {
		var p domain.Procurement
		if err := rows.Scan(&p.List, &p.ID, &p.Notice, &p.Process, &p.ProcessKey, &p.Modality, &p.Criterion, &p.OpensAt, &p.Object, &p.Status, &p.URL); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProcurementRepo) contractsByKeys(ctx context.Context, keys []string) ([]domain.ProcurementContract, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT procurement_id, notice, process, process_key, modality, object, value_cents, supplier, instrument, document_url
		FROM procurement_contracts WHERE process_key = ANY($1) ORDER BY procurement_id DESC, position`, pq.Array(keys))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ProcurementContract
	for rows.Next() {
		var c domain.ProcurementContract
		if err := rows.Scan(&c.ProcurementID, &c.Notice, &c.Process, &c.ProcessKey, &c.Modality, &c.Object, &c.ValueCents, &c.Supplier,
			&c.Instrument, &c.DocumentURL); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
