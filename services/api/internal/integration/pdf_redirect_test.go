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
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/nostore"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
	httpapi "github.com/seu-usuario/diario-sg/services/api/internal/presentation/http"
	"github.com/seu-usuario/diario-sg/services/api/migrations"
)

func TestGazettePDFRedirectsToOfficialFileWithoutBucket(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	ctx := context.Background()
	db := openTestDB(t, ctx, url)
	defer db.Close()
	if _, err := postgres.Migrate(ctx, db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, resetTables); err != nil {
		t.Fatal(err)
	}
	gaz := postgres.NewGazetteRepo(db)
	pub := &recPub{}
	in := usecase.IndexGazetteInput{EditionNumber: "1", PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
		SourceURL: "https://oficial.exemplo/1.pdf", StoragePath: "1.pdf", Checksum: strings.Repeat("a", 64)}
	if err := usecase.NewIndexGazette(gaz, textStore{}, passthroughExtractor{}, parser.Set{}, entities.New(), pub).Execute(ctx, in); err != nil {
		t.Fatal(err)
	}
	api := &httpapi.API{PDF: usecase.NewGetGazettePDF(gaz, nostore.Store{}), Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	srv := httptest.NewServer(api.Routes())
	defer srv.Close()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	resp, err := client.Get(srv.URL + "/v1/gazettes/" + pub.ids[0] + "/pdf")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != in.SourceURL {
		t.Fatalf("esperava 302 para o PDF oficial, veio %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if resp.Header.Get("ETag") != "" || strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Errorf("redirect não pode ser cacheado como o arquivo: %v", resp.Header)
	}
}
