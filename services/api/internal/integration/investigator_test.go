//go:build integration

package integration

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
	httpapi "github.com/seu-usuario/diario-sg/services/api/internal/presentation/http"
	mcpapi "github.com/seu-usuario/diario-sg/services/api/internal/presentation/mcp"
	"github.com/seu-usuario/diario-sg/services/api/migrations"
)

const investigatorGazette = "EXTRATO DO CONTRATO Nº 1/2026\nObjeto: limpeza urbana. Valor: R$ 1.000,00.\n" +
	"EXTRATO DO CONTRATO Nº 2/2026\nObjeto: coleta de lixo hospitalar. Valor: R$ 50.000,00.\n" +
	"EXTRATO DO CONTRATO Nº 3/2026\nObjeto: limpeza de escolas. Valor global R$ 200.000,00, mensal R$ 20.000,00."

func newInvestigatorServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv, _ := newInvestigatorServerWithDB(t)
	return srv
}

func newInvestigatorServerWithDB(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()
	return newServerFor(t, investigatorGazette)
}

func newServerFor(t *testing.T, text string) (*httptest.Server, *sql.DB) {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	ctx := context.Background()
	db := openTestDB(t, ctx, dbURL)
	t.Cleanup(func() { db.Close() })
	if _, err := postgres.Migrate(ctx, db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, resetTables); err != nil {
		t.Fatal(err)
	}
	gaz, acts := postgres.NewGazetteRepo(db), postgres.NewActRepo(db)
	in := usecase.IndexGazetteInput{EditionNumber: "9", PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
		SourceURL: "https://exemplo/9.pdf", StoragePath: "9.pdf", Checksum: strings.Repeat("f", 64)}
	idx := usecase.NewIndexGazette(gaz, stringStore(text), passthroughExtractor{}, parser.New(), entities.New(), &recPub{})
	if err := idx.Execute(ctx, in); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	keys := usecase.NewAPIKeys(postgres.NewAPIKeyRepo(db))
	mcpHandler := mcpapi.NewHandler(mcpapi.Deps{Search: usecase.NewSearchActs(acts), Read: usecase.NewReadAct(gaz, acts),
		Company: usecase.NewGetCompany(acts), Coverage: usecase.NewSourceCoverage(gaz), Keys: keys,
		PublicWebURL: "https://web.exemplo", Log: log})
	api := &httpapi.API{Search: usecase.NewSearchActs(acts), Stats: usecase.NewActStats(acts),
		Gazette: usecase.NewGetGazette(gaz, acts), Reports: usecase.NewErrorReports(postgres.NewErrorReportRepo(db)),
		Export: usecase.NewExportActs(acts), Feed: usecase.NewActFeed(acts), Organs: usecase.NewListOrgans(acts), PublicWebURL: "https://web.exemplo",
		Keys: keys, MCP: mcpHandler, Log: log}
	srv := httptest.NewServer(api.Routes())
	t.Cleanup(srv.Close)
	return srv, db
}

type valueHits struct {
	Items []struct {
		Title       string  `json:"title"`
		ValuesCents []int64 `json:"values_cents"`
	} `json:"items"`
}

func contractNumbers(t *testing.T, srv *httptest.Server, query string) []string {
	t.Helper()
	var hits valueHits
	getJSON(t, srv.URL+"/v1/acts?"+query, &hits)
	var out []string
	for _, h := range hits.Items {
		out = append(out, strings.TrimSuffix(strings.TrimPrefix(h.Title, "EXTRATO DO CONTRATO Nº "), "/2026"))
	}
	sort.Strings(out)
	return out
}

func TestSearchByValueRange(t *testing.T) {
	srv := newInvestigatorServer(t)

	cases := map[string][]string{
		"min_value=40000":                  {"2", "3"},
		"max_value=10000":                  {"1"},
		"min_value=15000&max_value=30000":  {"3"},
		"min_value=1000.00&max_value=1000": {"1"},
	}
	for query, want := range cases {
		if got := contractNumbers(t, srv, query); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: esperava %v, veio %v", query, want, got)
		}
	}

	var hits valueHits
	getJSON(t, srv.URL+"/v1/acts?min_value=15000&max_value=30000", &hits)
	if !reflect.DeepEqual(hits.Items[0].ValuesCents, []int64{20000000, 2000000}) {
		t.Errorf("valores citados inesperados: %v", hits.Items[0].ValuesCents)
	}

	var stats struct{ Items []struct{ Count int } }
	getJSON(t, srv.URL+"/v1/stats/acts?min_value=40000", &stats)
	if len(stats.Items) != 1 || stats.Items[0].Count != 2 {
		t.Errorf("estatísticas devem respeitar a faixa: %+v", stats)
	}

	for _, bad := range []string{"min_value=abc", "min_value=500&max_value=100", "max_value=1.000,00"} {
		r, err := http.Get(srv.URL + "/v1/acts?" + bad)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		if r.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: esperava 400, veio %d", bad, r.StatusCode)
		}
	}
}

func TestSearchOperators(t *testing.T) {
	srv := newInvestigatorServer(t)

	cases := map[string][]string{
		"limpeza OU coleta": {"1", "2", "3"},
		"limpeza OR coleta": {"1", "2", "3"},
		"limpeza ou coleta": {},
		"limpeza -urbana":   {"3"},
	}
	for q, want := range cases {
		got := contractNumbers(t, srv, "q="+url.QueryEscape(q))
		if len(got) == 0 && len(want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%q: esperava %v, veio %v", q, want, got)
		}
	}
}
