package postgres

import (
	"context"
	"database/sql"
	"fmt"

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
