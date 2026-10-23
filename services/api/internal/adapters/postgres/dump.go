package postgres

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type dumpTable struct {
	header  []string
	columns []string
	from    string
	order   string
}

func (t dumpTable) query() string {
	return "SELECT " + strings.Join(t.columns, ", ") + " " + t.from + " ORDER BY " + t.order
}

func (t dumpTable) queryForGazettes() string {
	return "SELECT g.id::text, " + strings.Join(t.columns, ", ") + " " + t.from +
		" WHERE g.id = ANY($1::uuid[]) ORDER BY " + t.order
}

func (t dumpTable) fingerprintAggregate(where string) string {
	return "coalesce((SELECT string_agg(concat_ws(E'\\x1f', " + strings.Join(t.columns, ", ") + "), E'\\x1e' ORDER BY " +
		t.order + ") " + t.from + " WHERE " + where + "), '')"
}

var dumpTables = []string{"gazettes", "acts", "act_entities"}

var dumpQueries = map[string]dumpTable{
	"gazettes": {
		header: []string{"id", "edition_number", "published_at", "is_extra", "source_url", "pdf_sha256", "indexed_at", "source"},
		columns: []string{"id::text", "edition_number", "to_char(published_at, 'YYYY-MM-DD')", "is_extra::text", "source_url", "checksum",
			`to_char(indexed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`, "source"},
		from:  "FROM gazettes",
		order: "published_at, source_url, id",
	},
	"acts": {
		header: []string{"id", "gazette_id", "position", "type", "organ", "title", "page_start", "page_end", "body"},
		columns: []string{"a.id::text", "a.gazette_id::text", "a.position::text", "a.type", "a.organ", "a.title",
			"coalesce(a.page_start::text, '')", "coalesce(a.page_end::text, '')", "a.body"},
		from:  "FROM acts a JOIN gazettes g ON g.id = a.gazette_id",
		order: "g.published_at, g.source_url, g.id, a.position",
	},
	"act_entities": {
		header:  []string{"act_id", "kind", "value", "normalized"},
		columns: []string{"e.act_id::text", "e.kind", "e.value", "e.normalized"},
		from:    "FROM act_entities e JOIN acts a ON a.id = e.act_id JOIN gazettes g ON g.id = a.gazette_id",
		order:   "g.published_at, g.source_url, g.id, a.position, e.kind, e.normalized",
	},
}

var fingerprintsQuery = "SELECT gz.id::text, md5(" +
	dumpQueries["acts"].fingerprintAggregate("g.id = gz.id") + " || E'\\x1d' || " +
	dumpQueries["act_entities"].fingerprintAggregate("g.id = gz.id") +
	") FROM gazettes gz ORDER BY gz.published_at, gz.source_url, gz.id"

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

func (s *dumpSnapshot) Header(table string) ([]string, error) {
	t, ok := dumpQueries[table]
	if !ok {
		return nil, fmt.Errorf("tabela fora do dump: %s", table)
	}
	return t.header, nil
}

func (s *dumpSnapshot) Fingerprints(ctx context.Context) ([]domain.GazetteFingerprint, error) {
	rows, err := s.tx.QueryContext(ctx, fingerprintsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.GazetteFingerprint
	for rows.Next() {
		var f domain.GazetteFingerprint
		if err := rows.Scan(&f.GazetteID, &f.Hash); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *dumpSnapshot) Records(ctx context.Context, table string, gazetteIDs []string) ([]domain.DumpRecord, error) {
	t, ok := dumpQueries[table]
	if !ok || table == "gazettes" {
		return nil, fmt.Errorf("tabela sem recorte por edição: %s", table)
	}
	rows, err := s.tx.QueryContext(ctx, t.queryForGazettes(), pq.Array(gazetteIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.DumpRecord
	for rows.Next() {
		r := domain.DumpRecord{Fields: make([]string, len(t.header))}
		dest := []any{&r.GazetteID}
		for i := range r.Fields {
			dest = append(dest, &r.Fields[i])
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *dumpSnapshot) WriteTable(ctx context.Context, table string, w io.Writer) (int, error) {
	t, ok := dumpQueries[table]
	if !ok {
		return 0, fmt.Errorf("tabela fora do dump: %s", table)
	}
	rows, err := s.tx.QueryContext(ctx, t.query())
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
