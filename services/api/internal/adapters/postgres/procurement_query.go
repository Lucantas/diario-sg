package postgres

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const (
	procurementPageSize = 20
	procurementColumns  = `list, id, notice, process, process_key, modality, criterion, opens_at, object, status, url`
	contractColumns     = `procurement_id, notice, process, process_key, modality, object, value_cents, supplier, instrument, document_url`
)

func (r *ProcurementRepo) QueryProcurements(ctx context.Context, q domain.ProcurementQuery) (domain.ProcurementReport, error) {
	var out domain.ProcurementReport
	var keys []string
	if q.CNPJ != "" {
		var err error
		if keys, err = r.commitmentProcessKeys(ctx, q.CNPJ); err != nil || len(keys) == 0 {
			return out, err
		}
	}
	tenders := procurementFilter(q, keys, `unaccent_immutable(object)`, `extract(year FROM opens_at)::int`)
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM procurements`+tenders.where(), tenders.args...).Scan(&out.ProcurementsTotal); err != nil {
		return out, err
	}
	var err error
	if out.Procurements, err = r.procurementPage(ctx, tenders, q.Offset); err != nil {
		return out, err
	}
	contracts := procurementFilter(q, keys, `unaccent_immutable(object || ' ' || supplier)`, `nullif(split_part(process_key, '/', 2), '')::int`)
	if err := r.db.QueryRowContext(ctx, `SELECT count(*), coalesce(sum(value_cents), 0) FROM procurement_contracts`+contracts.where(), contracts.args...).
		Scan(&out.ContractsTotal, &out.ContractValueCents); err != nil {
		return out, err
	}
	out.Contracts, err = r.contractPage(ctx, contracts, q.Offset)
	return out, err
}

func procurementFilter(q domain.ProcurementQuery, keys []string, textColumn, yearColumn string) sqlFilter {
	var f sqlFilter
	if q.CNPJ != "" {
		f.add(`process_key = ANY($?)`, pq.Array(keys))
	}
	if q.ProcessKey != "" {
		f.add(`process_key = $?`, q.ProcessKey)
	}
	if q.Text != "" {
		f.add(textColumn+` ILIKE unaccent_immutable($?)`, likePattern(q.Text))
	}
	if q.Organ != "" {
		f.add(`upper(notice) ~ $?`, `(^|[^A-Z])`+q.Organ+`($|[^A-Z])`)
	}
	if q.Years.From != 0 {
		f.add(yearColumn+` >= $?`, q.Years.From)
	}
	if q.Years.To != 0 {
		f.add(yearColumn+` <= $?`, q.Years.To)
	}
	return f
}

func (r *ProcurementRepo) procurementPage(ctx context.Context, f sqlFilter, offset int) ([]domain.Procurement, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+procurementColumns+` FROM procurements`+f.where()+`
		ORDER BY opens_at DESC NULLS LAST, id DESC LIMIT `+f.next()+` OFFSET $`+strconv.Itoa(len(f.args)+2),
		append(append([]any{}, f.args...), procurementPageSize, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProcurements(rows)
}

func (r *ProcurementRepo) contractPage(ctx context.Context, f sqlFilter, offset int) ([]domain.ProcurementContract, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+contractColumns+` FROM procurement_contracts`+f.where()+`
		ORDER BY procurement_id DESC, position LIMIT `+f.next()+` OFFSET $`+strconv.Itoa(len(f.args)+2),
		append(append([]any{}, f.args...), procurementPageSize, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContracts(rows)
}

func scanProcurements(rows *sql.Rows) ([]domain.Procurement, error) {
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

func scanContracts(rows *sql.Rows) ([]domain.ProcurementContract, error) {
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
