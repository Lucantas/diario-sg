package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const (
	paymentPageSize     = 20
	paymentTopSuppliers = 10
	spendingSums        = `count(*), coalesce(sum(committed_cents), 0), coalesce(sum(liquidated_cents), 0), coalesce(sum(paid_cents), 0)`
)

type sqlFilter struct {
	conds []string
	args  []any
}

func (f *sqlFilter) add(cond string, arg any) {
	f.args = append(f.args, arg)
	f.conds = append(f.conds, strings.ReplaceAll(cond, "$?", "$"+strconv.Itoa(len(f.args))))
}

func (f *sqlFilter) where() string {
	if len(f.conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(f.conds, " AND ")
}

func (f *sqlFilter) next() string { return "$" + strconv.Itoa(len(f.args)+1) }

func (r *MunicipalCommitmentRepo) QueryPayments(ctx context.Context, q domain.PaymentQuery) (domain.PaymentReport, error) {
	var out domain.PaymentReport
	f, err := r.paymentFilter(ctx, q)
	if err != nil {
		return out, err
	}
	w := f.where()
	if err := r.db.QueryRowContext(ctx, `SELECT `+spendingSums+` FROM municipal_commitments`+w, f.args...).
		Scan(&out.Totals.Commitments, &out.Totals.CommittedCents, &out.Totals.LiquidatedCents, &out.Totals.PaidCents); err != nil {
		return out, err
	}
	if out.Totals.Commitments == 0 {
		return out, nil
	}
	if out.ByYear, err = r.paymentYears(ctx, w, f.args); err != nil {
		return out, err
	}
	if out.ByEntity, err = r.paymentGroups(ctx, `entity_id::text, max(entity)`, `entity_id`, w, f.args, 0); err != nil {
		return out, err
	}
	if out.BySupplier, err = r.paymentGroups(ctx, `cnpj, max(name)`, `cnpj`, w, f.args, paymentTopSuppliers); err != nil {
		return out, err
	}
	out.Commitments, err = r.paymentPage(ctx, f, q.Offset)
	return out, err
}

func (r *MunicipalCommitmentRepo) paymentFilter(ctx context.Context, q domain.PaymentQuery) (sqlFilter, error) {
	var f sqlFilter
	if q.CNPJ != "" {
		f.add(`cnpj = $?`, q.CNPJ)
	}
	if q.Entity != "" {
		f.add(`unaccent_immutable(entity) ILIKE unaccent_immutable($?)`, likePattern(q.Entity))
	}
	if q.Text != "" {
		f.add(`(unaccent_immutable(object) ILIKE unaccent_immutable($?) OR unaccent_immutable(name) ILIKE unaccent_immutable($?))`, likePattern(q.Text))
	}
	if q.Years.From != 0 {
		f.add(`year >= $?`, q.Years.From)
	}
	if q.Years.To != 0 {
		f.add(`year <= $?`, q.Years.To)
	}
	if q.ProcessKey != "" {
		raws, err := r.processSpellings(ctx, q.ProcessKey)
		if err != nil {
			return f, err
		}
		f.add(`process = ANY($?)`, pq.Array(raws))
	}
	return f, nil
}

func (r *MunicipalCommitmentRepo) processSpellings(ctx context.Context, key string) ([]string, error) {
	_, year, _ := strings.Cut(key, "/")
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT process FROM municipal_commitments WHERE process LIKE '%' || $1 || '%'`, year[len(year)-2:])
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		if domain.ProcessKey(p) == key {
			out = append(out, p)
		}
	}
	return out, rows.Err()
}

func (r *MunicipalCommitmentRepo) paymentYears(ctx context.Context, where string, args []any) ([]domain.SpendingYear, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT year, `+spendingSums+` FROM municipal_commitments`+where+` GROUP BY year ORDER BY year DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SpendingYear
	for rows.Next() {
		var y domain.SpendingYear
		if err := rows.Scan(&y.Year, &y.Commitments, &y.CommittedCents, &y.LiquidatedCents, &y.PaidCents); err != nil {
			return nil, err
		}
		out = append(out, y)
	}
	return out, rows.Err()
}

func (r *MunicipalCommitmentRepo) paymentGroups(ctx context.Context, keyAndName, groupBy, where string, args []any, limit int) ([]domain.SpendingGroup, error) {
	query := `SELECT ` + keyAndName + `, ` + spendingSums + ` FROM municipal_commitments` + where +
		` GROUP BY ` + groupBy + ` ORDER BY 6 DESC, 1`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SpendingGroup
	for rows.Next() {
		var g domain.SpendingGroup
		if err := rows.Scan(&g.Key, &g.Name, &g.Commitments, &g.CommittedCents, &g.LiquidatedCents, &g.PaidCents); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *MunicipalCommitmentRepo) paymentPage(ctx context.Context, f sqlFilter, offset int) ([]domain.MunicipalCommitment, error) {
	limit, skip := f.next(), "$"+strconv.Itoa(len(f.args)+2)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+municipalCommitmentColumns+` FROM municipal_commitments`+f.where()+`
		ORDER BY committed_on DESC, year DESC, entity_id, commitment_id DESC LIMIT `+limit+` OFFSET `+skip,
		append(append([]any{}, f.args...), paymentPageSize, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMunicipalCommitments(rows)
}

func scanMunicipalCommitments(rows *sql.Rows) ([]domain.MunicipalCommitment, error) {
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
