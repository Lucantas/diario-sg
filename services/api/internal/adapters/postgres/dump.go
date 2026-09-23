package postgres

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type dumpTable struct {
	header []string
	query  string
}

var dumpTables = []string{"gazettes", "acts", "act_entities"}

var dumpQueries = map[string]dumpTable{
	"gazettes": {
		header: []string{"id", "edition_number", "published_at", "is_extra", "source_url", "pdf_sha256", "indexed_at"},
		query: `SELECT id::text, edition_number, to_char(published_at, 'YYYY-MM-DD'), is_extra::text, source_url, checksum,
		               to_char(indexed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		        FROM gazettes ORDER BY published_at, source_url`,
	},
	"acts": {
		header: []string{"id", "gazette_id", "position", "type", "organ", "title", "page_start", "page_end", "body"},
		query: `SELECT a.id::text, a.gazette_id::text, a.position::text, a.type, a.organ, a.title,
		               coalesce(a.page_start::text, ''), coalesce(a.page_end::text, ''), a.body
		        FROM acts a JOIN gazettes g ON g.id = a.gazette_id
		        ORDER BY g.published_at, g.source_url, a.position`,
	},
	"act_entities": {
		header: []string{"act_id", "kind", "value", "normalized"},
		query: `SELECT e.act_id::text, e.kind, e.value, e.normalized
		        FROM act_entities e JOIN acts a ON a.id = e.act_id JOIN gazettes g ON g.id = a.gazette_id
		        ORDER BY g.published_at, g.source_url, a.position, e.kind, e.normalized`,
	},
}

type DumpSource struct{ db *sql.DB }

func NewDumpSource(db *sql.DB) *DumpSource { return &DumpSource{db: db} }

func (s *DumpSource) Snapshot(ctx context.Context) (ports.DumpSnapshot, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	return &dumpSnapshot{tx: tx}, nil
}

type dumpSnapshot struct{ tx *sql.Tx }

func (s *dumpSnapshot) Tables() []string { return dumpTables }

func (s *dumpSnapshot) Close() error { return s.tx.Rollback() }

func (s *dumpSnapshot) WriteTable(ctx context.Context, table string, w io.Writer) (int, error) {
	t, ok := dumpQueries[table]
	if !ok {
		return 0, fmt.Errorf("tabela fora do dump: %s", table)
	}
	rows, err := s.tx.QueryContext(ctx, t.query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	out := csv.NewWriter(w)
	if err := out.Write(t.header); err != nil {
		return 0, err
	}
	record := make([]string, len(t.header))
	dest := make([]any, len(t.header))
	for i := range record {
		dest[i] = &record[i]
	}
	n := 0
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return n, err
		}
		if err := out.Write(record); err != nil {
			return n, err
		}
		n++
	}
	if err := rows.Err(); err != nil {
		return n, err
	}
	out.Flush()
	return n, out.Error()
}
