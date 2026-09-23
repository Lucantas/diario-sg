//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
)

func TestDumpSourceWritesTheThreeTables(t *testing.T) {
	_, db := newInvestigatorServerWithDB(t)
	ctx := context.Background()
	snap, err := postgres.NewDumpSource(db).Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	if got := strings.Join(snap.Tables(), ","); got != "gazettes,acts,act_entities" {
		t.Fatalf("tabelas inesperadas: %s", got)
	}
	headers := map[string]string{
		"gazettes":     "id,edition_number,published_at,is_extra,source_url,pdf_sha256,indexed_at",
		"acts":         "id,gazette_id,position,type,organ,title,page_start,page_end,body",
		"act_entities": "act_id,kind,value,normalized",
	}
	records := map[string][][]string{}
	for table, header := range headers {
		var buf bytes.Buffer
		rows, err := snap.WriteTable(ctx, table, &buf)
		if err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		recs, err := csv.NewReader(&buf).ReadAll()
		if err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		if strings.Join(recs[0], ",") != header || rows != len(recs)-1 {
			t.Fatalf("%s: cabeçalho %q, %d linhas informadas, %d no CSV", table, recs[0], rows, len(recs)-1)
		}
		records[table] = recs[1:]
	}

	g := records["gazettes"]
	if len(g) != 1 || g[0][2] != "2026-09-18" || g[0][3] != "false" || g[0][5] != strings.Repeat("f", 64) {
		t.Fatalf("edições inesperadas: %q", g)
	}
	a := records["acts"]
	if len(a) != 3 || a[0][2] != "0" || a[0][3] != "contrato" || a[0][6] != "1" || !strings.Contains(a[2][8], "mensal R$ 20.000,00") {
		t.Fatalf("atos inesperados: %q", a)
	}
	valores := 0
	for _, e := range records["act_entities"] {
		if e[1] == "valor" {
			valores++
		}
	}
	if valores != 4 {
		t.Fatalf("esperava 4 valores, veio %d: %q", valores, records["act_entities"])
	}

	if _, err := snap.WriteTable(ctx, "subscriptions", &bytes.Buffer{}); err == nil {
		t.Fatal("tabela fora do dump deve dar erro")
	}
}
