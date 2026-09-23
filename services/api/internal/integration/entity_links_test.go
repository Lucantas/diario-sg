//go:build integration

package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const resetTables = `TRUNCATE gazettes, subscriptions, api_keys, entities, fetch_runs CASCADE`

type link struct {
	kind, key, certainty, evidence string
}

func diarioLinks(t *testing.T, db *sql.DB) map[string]link {
	t.Helper()
	rows, err := db.Query(`
		SELECT e.kind, e.key, l.certainty, l.evidence, l.record_id
		FROM entity_links l JOIN entities e ON e.id = l.entity_id
		WHERE l.source = $1 AND l.record_kind = $2 AND l.role = $3`,
		domain.SourceDiarioPrefeitura, domain.RecordAct, domain.RoleMentioned)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]link{}
	for rows.Next() {
		var l link
		var record string
		if err := rows.Scan(&l.kind, &l.key, &l.certainty, &l.evidence, &record); err != nil {
			t.Fatal(err)
		}
		out[l.kind+":"+l.key] = l
	}
	return out
}

func orphanLinks(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`
		SELECT count(*) FROM entity_links l
		WHERE l.source = $1 AND NOT EXISTS (SELECT 1 FROM acts a WHERE a.id::text = l.record_id)`,
		domain.SourceDiarioPrefeitura).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestIndexingLinksActsToEntities(t *testing.T) {
	_, db := newServerFor(t, gazetteText)
	ctx := context.Background()

	got := diarioLinks(t, db)
	want := map[string]string{
		"cnpj:12345678000190": "exata",
		"processo:81892025":   "forte",
		"contrato:55/2026":    "fraca",
	}
	for k, certainty := range want {
		if got[k].certainty != certainty || got[k].evidence == "" {
			t.Errorf("%s: esperava certeza %s com evidência, veio %+v", k, certainty, got[k])
		}
	}
	var fromExtraction int
	if err := db.QueryRow(`
		SELECT count(DISTINCT (act_id, kind, entity_key(kind, normalized)))
		FROM act_entities WHERE kind IN ('cnpj', 'processo', 'contrato')`).Scan(&fromExtraction); err != nil {
		t.Fatal(err)
	}
	var links int
	if err := db.QueryRow(`SELECT count(*) FROM entity_links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if links != fromExtraction || links == 0 {
		t.Fatalf("uma ligação por ato e chave: %d ligações, %d na extração", links, fromExtraction)
	}

	gaz := postgres.NewGazetteRepo(db)
	published := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	res, err := usecase.NewReindexGazettes(gaz, stringStore(gazetteText), passthroughExtractor{}, parser.New(), entities.New()).
		Execute(ctx, published, published)
	if err != nil || res.Reindexed != 1 {
		t.Fatalf("reindexação: %+v %v", res, err)
	}
	if n := orphanLinks(t, db); n != 0 {
		t.Fatalf("reindexar deixou %d ligações órfãs", n)
	}
	if after := diarioLinks(t, db); len(after) != len(got) {
		t.Fatalf("reindexar mudou as ligações: antes %d, depois %d", len(got), len(after))
	}
}

func TestSQLLinkRuleMatchesTheDomain(t *testing.T) {
	_, db := newServerFor(t, gazetteText)
	cases := [][2]string{
		{"cnpj", "28636579000100"}, {"processo", "061098120257"}, {"contrato", "001/2017"},
		{"contrato", "30/FMS/2011"}, {"contrato", "0/2020"}, {"contrato", "007/2024/SEMAD"},
	}
	rows, err := db.Query(`SELECT kind, normalized FROM act_entities WHERE kind IN ('cnpj', 'processo', 'contrato')`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var c [2]string
		if err := rows.Scan(&c[0], &c[1]); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, c)
	}
	rows.Close()
	if len(cases) <= 6 {
		t.Fatal("a edição de teste deveria ter entidades extraídas")
	}

	for _, c := range cases {
		kind := domain.EntityKind(c[0])
		var key, certainty string
		if err := db.QueryRow(`SELECT entity_key($1, $2), link_certainty($1, entity_key($1, $2))`, c[0], c[1]).
			Scan(&key, &certainty); err != nil {
			t.Fatal(err)
		}
		goKey := domain.EntityKey(kind, c[1])
		if key != goKey || certainty != string(domain.LinkCertainty(kind, goKey)) {
			t.Errorf("%s %q: SQL deu %q/%s, Go deu %q/%s", c[0], c[1], key, certainty, goKey, domain.LinkCertainty(kind, goKey))
		}
	}
}
