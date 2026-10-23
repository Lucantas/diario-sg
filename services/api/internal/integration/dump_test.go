//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
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
		"gazettes":     "id,edition_number,published_at,is_extra,source_url,pdf_sha256,indexed_at,source",
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

func TestDumpFingerprintsMatchTheFingerprintOfTheCSVRows(t *testing.T) {
	_, db := newInvestigatorServerWithDB(t)
	ctx := context.Background()
	snap, err := postgres.NewDumpSource(db).Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	fingerprints, err := snap.Fingerprints(ctx)
	if err != nil {
		t.Fatal(err)
	}
	acts := csvRows(t, snap, "acts")
	entities := csvRows(t, snap, "act_entities")

	if len(fingerprints) != 1 || fingerprints[0].Hash != domain.FingerprintOf(acts, entities) {
		t.Fatalf("impressão digital do banco difere da do CSV: %+v", fingerprints)
	}
}

func TestDumpRecordsReturnsOnlyTheRequestedGazettesInDumpOrder(t *testing.T) {
	_, db := newInvestigatorServerWithDB(t)
	ctx := context.Background()
	snap, err := postgres.NewDumpSource(db).Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()
	fingerprints, err := snap.Fingerprints(ctx)
	if err != nil {
		t.Fatal(err)
	}
	gazetteID := fingerprints[0].GazetteID

	for _, table := range []string{"acts", "act_entities"} {
		records, err := snap.Records(ctx, table, []string{gazetteID})
		if err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		var fields [][]string
		for _, r := range records {
			if r.GazetteID != gazetteID {
				t.Fatalf("%s: edição inesperada %s", table, r.GazetteID)
			}
			fields = append(fields, r.Fields)
		}
		if want := csvRows(t, snap, table); !reflect.DeepEqual(fields, want) {
			t.Fatalf("%s: recorte difere do CSV completo:\n%q\n%q", table, fields, want)
		}
		none, err := snap.Records(ctx, table, []string{"00000000-0000-0000-0000-000000000000"})
		if err != nil || len(none) != 0 {
			t.Fatalf("%s: edição inexistente deveria vir vazia: %v %v", table, none, err)
		}
	}
	if _, err := snap.Records(ctx, "gazettes", []string{gazetteID}); err == nil {
		t.Fatal("gazettes não tem recorte por edição")
	}
}

func csvRows(t *testing.T, snap interface {
	WriteTable(context.Context, string, io.Writer) (int, error)
}, table string) [][]string {
	t.Helper()
	var buf bytes.Buffer
	if _, err := snap.WriteTable(context.Background(), table, &buf); err != nil {
		t.Fatal(err)
	}
	recs, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return recs[1:]
}
