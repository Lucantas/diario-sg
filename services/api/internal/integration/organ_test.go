//go:build integration

package integration

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
	httpapi "github.com/seu-usuario/diario-sg/services/api/internal/presentation/http"
	"github.com/seu-usuario/diario-sg/services/api/migrations"
)

const organGazette = "ATOS DO PREFEITO\nDECRETO Nº 1/2026\nDispõe sobre o horário.\n" +
	"SEMAD\nPORTARIA Nº 10/2026\nNomeia servidor para a função.\n" +
	"FMS\nEXTRATO DO CONTRATO Nº 3/2026\nObjeto: medicamentos."

type stringStore string

func (s stringStore) Get(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(string(s))), nil
}

func newOrganServer(t *testing.T) *httptest.Server {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	ctx := context.Background()
	db := openTestDB(t, ctx, url)
	t.Cleanup(func() { db.Close() })
	if _, err := postgres.Migrate(ctx, db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE gazettes, subscriptions CASCADE`); err != nil {
		t.Fatal(err)
	}
	gaz, acts := postgres.NewGazetteRepo(db), postgres.NewActRepo(db)
	in := usecase.IndexGazetteInput{EditionNumber: "7", PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
		SourceURL: "https://exemplo/7.pdf", StoragePath: "7.pdf", Checksum: strings.Repeat("e", 64)}
	idx := usecase.NewIndexGazette(gaz, stringStore(organGazette), passthroughExtractor{}, parser.New(), entities.New(), &recPub{})
	if err := idx.Execute(ctx, in); err != nil {
		t.Fatal(err)
	}
	api := &httpapi.API{Search: usecase.NewSearchActs(acts), Gazette: usecase.NewGetGazette(gaz, acts),
		Company: usecase.NewGetCompany(acts), Stats: usecase.NewActStats(acts), Organs: usecase.NewListOrgans(acts),
		Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	srv := httptest.NewServer(api.Routes())
	t.Cleanup(srv.Close)
	return srv
}

type organHits struct {
	Items []struct {
		Title     string `json:"title"`
		Organ     string `json:"organ"`
		OrganName string `json:"organ_name"`
	} `json:"items"`
	Total int `json:"total"`
}

func TestSearchFiltersByOrgan(t *testing.T) {
	srv := newOrganServer(t)

	var all organHits
	getJSON(t, srv.URL+"/v1/acts", &all)
	if all.Total != 3 {
		t.Fatalf("esperava 3 atos, veio %+v", all)
	}
	for _, it := range all.Items {
		if strings.HasPrefix(it.Title, "DECRETO") && it.Organ != "" {
			t.Fatalf("decreto antes da primeira sigla não tem órgão: %+v", it)
		}
	}

	var semad organHits
	getJSON(t, srv.URL+"/v1/acts?organ=semad", &semad)
	if semad.Total != 1 || semad.Items[0].Organ != "SEMAD" || semad.Items[0].OrganName != domain.OrganName("SEMAD") {
		t.Fatalf("filtro por órgão inesperado: %+v", semad)
	}

	if r, err := http.Get(srv.URL + "/v1/acts?organ=TOTAL"); err != nil || r.StatusCode != http.StatusBadRequest {
		t.Fatalf("órgão desconhecido deve dar 400: %v %v", r, err)
	}

	var stats struct {
		Items []struct{ Count int }
	}
	getJSON(t, srv.URL+"/v1/stats/acts?organ=FMS", &stats)
	if len(stats.Items) != 1 || stats.Items[0].Count != 1 {
		t.Fatalf("estatísticas devem respeitar o órgão: %+v", stats)
	}
}

func TestListOrgansCountsActs(t *testing.T) {
	srv := newOrganServer(t)

	var organs struct {
		Items []struct {
			Acronym string `json:"acronym"`
			Name    string `json:"name"`
			Acts    int    `json:"acts"`
		} `json:"items"`
	}
	r, err := http.Get(srv.URL + "/v1/organs")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.Header.Get("Cache-Control") == "" {
		t.Error("lista de órgãos deve ter Cache-Control")
	}
	getJSON(t, srv.URL+"/v1/organs", &organs)

	if len(organs.Items) != 2 || organs.Items[0].Acronym != "FMS" || organs.Items[1].Acronym != "SEMAD" ||
		organs.Items[0].Acts != 1 || organs.Items[1].Acts != 1 || organs.Items[1].Name != domain.OrganName("SEMAD") {
		t.Fatalf("órgãos inesperados: %+v", organs)
	}
}
