package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type BillRepo struct{ db *sql.DB }

func NewBillRepo(db *sql.DB) *BillRepo { return &BillRepo{db: db} }

func (r *BillRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM bills, bill_events, bill_opinions LIMIT 0`)
	return err
}

func (r *BillRepo) KnownBills(ctx context.Context) (map[domain.BillKey]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT process_number, process_year FROM bills`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[domain.BillKey]bool{}
	for rows.Next() {
		var k domain.BillKey
		if err := rows.Scan(&k.Number, &k.Year); err != nil {
			return nil, err
		}
		out[k] = true
	}
	return out, rows.Err()
}

func (r *BillRepo) StaleOpenBills(ctx context.Context, kinds []string, limit int) ([]domain.BillKey, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT process_number, process_year FROM bills
		WHERE kind = ANY($1) AND status <> 'Arquivado' AND law_number = 0
		ORDER BY fetched_at, process_year, process_number LIMIT $2`, pq.StringArray(kinds), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.BillKey
	for rows.Next() {
		var k domain.BillKey
		if err := rows.Scan(&k.Number, &k.Year); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *BillRepo) SaveBills(ctx context.Context, bills []domain.Bill) error {
	if len(bills) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	numbers, years := make([]int64, len(bills)), make([]int64, len(bills))
	for i, b := range bills {
		if err := upsertBill(ctx, tx, b); err != nil {
			return fmt.Errorf("processo %s: %w", b.Key, err)
		}
		numbers[i], years[i] = int64(b.Key.Number), int64(b.Key.Year)
	}
	for _, table := range []string{"bill_events", "bill_opinions"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` t USING unnest($1::int[], $2::int[]) AS k(n, y)
			WHERE t.process_number = k.n AND t.process_year = k.y`, pq.Int64Array(numbers), pq.Int64Array(years)); err != nil {
			return err
		}
	}
	if err := copyBillEvents(ctx, tx, bills); err != nil {
		return err
	}
	if err := copyBillOpinions(ctx, tx, bills); err != nil {
		return err
	}
	return tx.Commit()
}

func upsertBill(ctx context.Context, tx *sql.Tx, b domain.Bill) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO bills (process_number, process_year, kind, doc_label, doc_number, doc_year, summary, authors,
			presented_on, status, current_body, last_movement, source_updated_at, law_number, law_year, law_url, url, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (process_number, process_year) DO UPDATE SET kind = EXCLUDED.kind, doc_label = EXCLUDED.doc_label,
			doc_number = EXCLUDED.doc_number, doc_year = EXCLUDED.doc_year, summary = EXCLUDED.summary, authors = EXCLUDED.authors,
			presented_on = EXCLUDED.presented_on, status = EXCLUDED.status, current_body = EXCLUDED.current_body,
			last_movement = EXCLUDED.last_movement, source_updated_at = EXCLUDED.source_updated_at, law_number = EXCLUDED.law_number,
			law_year = EXCLUDED.law_year, law_url = EXCLUDED.law_url, url = EXCLUDED.url, fetched_at = EXCLUDED.fetched_at`,
		b.Key.Number, b.Key.Year, b.Kind, b.DocLabel, b.DocNumber, b.DocYear, b.Summary, b.Authors, b.PresentedOn, b.Status,
		b.CurrentBody, b.LastMovement, b.SourceUpdatedAt, b.LawNumber, b.LawYear, b.LawURL, b.URL, b.FetchedAt)
	return err
}

func copyBillEvents(ctx context.Context, tx *sql.Tx, bills []domain.Bill) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("bill_events", "process_number", "process_year", "position", "happened_at", "label", "text", "sector"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for _, b := range bills {
		for _, e := range b.Events {
			if _, err := stmt.ExecContext(ctx, b.Key.Number, b.Key.Year, e.Position, e.At, e.Label, e.Text, e.Sector); err != nil {
				return fmt.Errorf("evento %d do processo %s: %w", e.Position, b.Key, err)
			}
		}
	}
	return closeCopy(ctx, stmt)
}

func copyBillOpinions(ctx context.Context, tx *sql.Tx, bills []domain.Bill) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("bill_opinions", "process_number", "process_year", "position", "result", "issued_on", "committee", "rapporteur"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for _, b := range bills {
		for _, o := range b.Opinions {
			if _, err := stmt.ExecContext(ctx, b.Key.Number, b.Key.Year, o.Position, o.Result, o.On, o.Committee, o.Rapporteur); err != nil {
				return fmt.Errorf("parecer %d do processo %s: %w", o.Position, b.Key, err)
			}
		}
	}
	return closeCopy(ctx, stmt)
}

const billColumns = `b.process_number, b.process_year, b.kind, b.doc_label, b.doc_number, b.doc_year, b.summary, b.authors, b.presented_on,
	b.status, b.current_body, b.last_movement, b.source_updated_at, b.law_number, b.law_year, b.law_url, b.url, b.fetched_at`

func (r *BillRepo) CandidateBills(ctx context.Context, f domain.BillFilter) ([]domain.Bill, error) {
	q, args := candidateBillsSQL(f)
	bills, err := r.queryBills(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return bills, r.attachEvents(ctx, bills)
}

func candidateBillsSQL(f domain.BillFilter) (string, []any) {
	var b strings.Builder
	var args []any
	param := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	b.WriteString(`SELECT ` + billColumns + ` FROM bills b WHERE true`)
	if len(f.Kinds) > 0 {
		b.WriteString(` AND b.kind = ANY(` + param(pq.StringArray(f.Kinds)) + `)`)
	}
	if f.Text != "" {
		b.WriteString(` AND to_tsvector('portuguese_unaccent', b.summary || ' ' || b.authors) @@ websearch_to_tsquery('portuguese_unaccent', ` + param(f.Text) + `)`)
	}
	if f.Author != "" {
		b.WriteString(` AND unaccent(b.authors) ILIKE '%' || unaccent(` + param(f.Author) + `) || '%'`)
	}
	if f.Status != "" {
		b.WriteString(` AND lower(unaccent(b.status)) = lower(unaccent(` + param(f.Status) + `))`)
	}
	if !f.From.IsZero() {
		b.WriteString(` AND b.presented_on >= ` + param(f.From.Format(time.DateOnly)) + `::date`)
	}
	if !f.To.IsZero() {
		b.WriteString(` AND b.presented_on <= ` + param(f.To.Format(time.DateOnly)) + `::date`)
	}
	if !f.Theme.IsZero() {
		committee := param("%comissao%" + f.Theme.Committee + "%")
		b.WriteString(` AND ((lower(unaccent(b.summary)) ~ ` + param(f.Theme.TermsSQLRegex()) + ` AND lower(unaccent(b.summary)) !~ ` + param(f.Theme.ExcludeSQLRegex()) + `)
			OR lower(unaccent(b.current_body)) LIKE ` + param("%"+f.Theme.Committee+"%") + `
			OR EXISTS (SELECT 1 FROM bill_opinions o WHERE o.process_number = b.process_number AND o.process_year = b.process_year AND lower(unaccent(o.committee)) LIKE ` + committee + `)
			OR EXISTS (SELECT 1 FROM bill_events e WHERE e.process_number = b.process_number AND e.process_year = b.process_year AND lower(unaccent(e.text)) LIKE ` + committee + `))`)
	}
	b.WriteString(` ORDER BY b.presented_on DESC NULLS LAST, b.process_year DESC, b.process_number DESC`)
	return b.String(), args
}

func (r *BillRepo) BillByKey(ctx context.Context, key domain.BillKey) (domain.Bill, bool, error) {
	bills, err := r.queryBills(ctx, `SELECT `+billColumns+` FROM bills b WHERE b.process_number = $1 AND b.process_year = $2`, key.Number, key.Year)
	if err != nil || len(bills) == 0 {
		return domain.Bill{}, false, err
	}
	if err := r.attachEvents(ctx, bills); err != nil {
		return domain.Bill{}, false, err
	}
	if err := r.attachOpinions(ctx, bills); err != nil {
		return domain.Bill{}, false, err
	}
	return bills[0], true, nil
}

func (r *BillRepo) BillsByDoc(ctx context.Context, ref domain.BillDocRef) ([]domain.Bill, error) {
	return r.queryBills(ctx, `SELECT `+billColumns+` FROM bills b WHERE b.kind = $1 AND b.doc_number = $2 AND b.doc_year = $3
		ORDER BY b.process_year, b.process_number`, ref.Kind, ref.Number, ref.Year)
}

func (r *BillRepo) BillsByLaw(ctx context.Context, number, year int) ([]domain.Bill, error) {
	return r.queryBills(ctx, `SELECT `+billColumns+` FROM bills b WHERE b.law_number = $1 AND b.law_year = $2
		ORDER BY b.process_year, b.process_number`, number, year)
}

func (r *BillRepo) queryBills(ctx context.Context, q string, args ...any) ([]domain.Bill, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Bill{}
	for rows.Next() {
		var b domain.Bill
		if err := rows.Scan(&b.Key.Number, &b.Key.Year, &b.Kind, &b.DocLabel, &b.DocNumber, &b.DocYear, &b.Summary, &b.Authors, &b.PresentedOn,
			&b.Status, &b.CurrentBody, &b.LastMovement, &b.SourceUpdatedAt, &b.LawNumber, &b.LawYear, &b.LawURL, &b.URL, &b.FetchedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func billKeyArrays(bills []domain.Bill) (pq.Int64Array, pq.Int64Array, map[domain.BillKey]int) {
	numbers, years := make(pq.Int64Array, len(bills)), make(pq.Int64Array, len(bills))
	index := make(map[domain.BillKey]int, len(bills))
	for i, b := range bills {
		numbers[i], years[i] = int64(b.Key.Number), int64(b.Key.Year)
		index[b.Key] = i
	}
	return numbers, years, index
}

func (r *BillRepo) attachEvents(ctx context.Context, bills []domain.Bill) error {
	if len(bills) == 0 {
		return nil
	}
	numbers, years, index := billKeyArrays(bills)
	rows, err := r.db.QueryContext(ctx, `SELECT e.process_number, e.process_year, e.position, e.happened_at, e.label, e.text, e.sector
		FROM bill_events e JOIN unnest($1::int[], $2::int[]) AS k(n, y) ON e.process_number = k.n AND e.process_year = k.y
		ORDER BY e.process_year, e.process_number, e.position`, numbers, years)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var k domain.BillKey
		var e domain.BillEvent
		if err := rows.Scan(&k.Number, &k.Year, &e.Position, &e.At, &e.Label, &e.Text, &e.Sector); err != nil {
			return err
		}
		i := index[k]
		bills[i].Events = append(bills[i].Events, e)
	}
	return rows.Err()
}

func (r *BillRepo) attachOpinions(ctx context.Context, bills []domain.Bill) error {
	numbers, years, index := billKeyArrays(bills)
	rows, err := r.db.QueryContext(ctx, `SELECT o.process_number, o.process_year, o.position, o.result, o.issued_on, o.committee, o.rapporteur
		FROM bill_opinions o JOIN unnest($1::int[], $2::int[]) AS k(n, y) ON o.process_number = k.n AND o.process_year = k.y
		ORDER BY o.process_year, o.process_number, o.position`, numbers, years)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var k domain.BillKey
		var o domain.BillOpinion
		if err := rows.Scan(&k.Number, &k.Year, &o.Position, &o.Result, &o.On, &o.Committee, &o.Rapporteur); err != nil {
			return err
		}
		i := index[k]
		bills[i].Opinions = append(bills[i].Opinions, o)
	}
	return rows.Err()
}
