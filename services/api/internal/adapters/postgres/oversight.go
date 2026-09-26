package postgres

import (
	"context"
	"database/sql"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const stalledWorkLinkRole = "contratada"

type OversightRepo struct{ db *sql.DB }

func NewOversightRepo(db *sql.DB) *OversightRepo { return &OversightRepo{db: db} }

func (r *OversightRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM tce_accounts, tce_penalties, tce_stalled_works LIMIT 0`)
	return err
}

func (r *OversightRepo) ReplaceOversight(ctx context.Context, o domain.TCEOversight) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `TRUNCATE tce_accounts, tce_penalties, tce_stalled_works`); err != nil {
		return err
	}
	for _, a := range o.Accounts {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tce_accounts (year, opinion, process, responsible) VALUES ($1, $2, $3, $4)
			ON CONFLICT (year) DO UPDATE SET opinion = excluded.opinion, process = excluded.process, responsible = excluded.responsible`,
			a.Year, a.Opinion, a.Process, a.Responsible); err != nil {
			return err
		}
	}
	for _, p := range o.Penalties {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tce_penalties (condemnation, process, year, value_cents, organ, nature, session_date)
			VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (condemnation) DO NOTHING`,
			p.Condemnation, p.Process, p.Year, p.ValueCents, p.Organ, p.Nature, p.SessionDate); err != nil {
			return err
		}
	}
	for _, w := range o.Works {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tce_stalled_works (contract, cnpj, contractor, organ, function, total_cents, paid_cents,
			stalled_at, started_at, stalled_for, reason, contract_status, funding) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			w.Contract, w.CNPJ, w.Contractor, w.Organ, w.Function, w.TotalCents, w.PaidCents, w.StalledAt, w.StartedAt, w.StalledFor,
			w.Reason, w.ContractStatus, w.Funding); err != nil {
			return err
		}
	}
	if err := relinkStalledWorks(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func relinkStalledWorks(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM entity_links WHERE source = $1`, domain.SourceOversight); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO entity_links (entity_id, source, record_kind, record_id, role, certainty, evidence)
		SELECT e.id, $1, $2, w.id::text, $3, $4, 'contrato ' || w.contract
		FROM tce_stalled_works w JOIN entities e ON e.kind = 'cnpj' AND e.key = w.cnpj
		ON CONFLICT DO NOTHING`, domain.SourceOversight, domain.RecordStalledWork, stalledWorkLinkRole, domain.CertaintyExact)
	return err
}

const stalledWorkColumns = `contract, cnpj, contractor, organ, function, total_cents, paid_cents, stalled_at, started_at, stalled_for,
	reason, contract_status, funding`

func scanStalledWorks(rows *sql.Rows) ([]domain.StalledWork, error) {
	out := []domain.StalledWork{}
	for rows.Next() {
		var w domain.StalledWork
		if err := rows.Scan(&w.Contract, &w.CNPJ, &w.Contractor, &w.Organ, &w.Function, &w.TotalCents, &w.PaidCents, &w.StalledAt,
			&w.StartedAt, &w.StalledFor, &w.Reason, &w.ContractStatus, &w.Funding); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *OversightRepo) StalledWorksByCNPJ(ctx context.Context, cnpj string) ([]domain.StalledWork, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+stalledWorkColumns+` FROM tce_stalled_works WHERE cnpj = $1 ORDER BY stalled_at DESC, id`, cnpj)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStalledWorks(rows)
}

func (r *OversightRepo) Oversight(ctx context.Context) (domain.TCEOversight, error) {
	var o domain.TCEOversight
	var err error
	if o.Accounts, err = r.accounts(ctx); err != nil {
		return o, err
	}
	if o.Penalties, err = r.penalties(ctx); err != nil {
		return o, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+stalledWorkColumns+` FROM tce_stalled_works ORDER BY stalled_at DESC, id`)
	if err != nil {
		return o, err
	}
	defer rows.Close()
	o.Works, err = scanStalledWorks(rows)
	return o, err
}

func (r *OversightRepo) accounts(ctx context.Context) ([]domain.TCEAccount, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT year, opinion, process, responsible FROM tce_accounts ORDER BY year DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TCEAccount{}
	for rows.Next() {
		var a domain.TCEAccount
		if err := rows.Scan(&a.Year, &a.Opinion, &a.Process, &a.Responsible); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *OversightRepo) penalties(ctx context.Context) ([]domain.TCEPenalty, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT condemnation, process, year, value_cents, organ, nature, session_date FROM tce_penalties
		ORDER BY session_date DESC NULLS LAST, process, condemnation`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TCEPenalty{}
	for rows.Next() {
		var p domain.TCEPenalty
		if err := rows.Scan(&p.Condemnation, &p.Process, &p.Year, &p.ValueCents, &p.Organ, &p.Nature, &p.SessionDate); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
