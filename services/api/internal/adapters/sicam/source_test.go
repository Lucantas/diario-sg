package sicam

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func sitemapServer(t *testing.T, agents *[]string) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*agents = append(*agents, r.UserAgent())
		switch r.URL.Path {
		case "/sitemap.xml":
			fmt.Fprintf(w, `<sitemapindex><sitemap><loc>%[1]s/sitemap-leis.xml</loc></sitemap>
				<sitemap><loc>%[1]s/sitemap-processos-1.xml</loc></sitemap><sitemap><loc>%[1]s/sitemap-processos-2.xml</loc></sitemap>
				<sitemap><loc>https://outro.exemplo/sitemap-processos-3.xml</loc></sitemap></sitemapindex>`, srv.URL)
		case "/sitemap-processos-1.xml":
			fmt.Fprint(w, `<urlset><url><loc>x/areapublica/processo/5564-2025</loc></url><url><loc>x/areapublica/processos</loc></url></urlset>`)
		case "/sitemap-processos-2.xml":
			fmt.Fprint(w, `<urlset><url><loc>x/areapublica/processo/211-2014</loc></url></urlset>`)
		case "/areapublica/processo/5564-2025":
			fmt.Fprint(w, `<html><main>página</main></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestProcessKeysReadsOnlyProcessSitemapsOfTheSameHost(t *testing.T) {
	var agents []string
	srv := sitemapServer(t, &agents)
	src := New(srv.URL+"/", srv.Client())
	src.pause = 0

	keys, err := src.ProcessKeys(context.Background())

	if err != nil || len(keys) != 2 || keys[0] != (domain.BillKey{Number: 5564, Year: 2025}) || keys[1] != (domain.BillKey{Number: 211, Year: 2014}) {
		t.Fatalf("chaves: %v %v", keys, err)
	}
	for _, a := range agents {
		if !strings.HasPrefix(a, "diario-sg-bot/") {
			t.Fatalf("User-Agent: %q", a)
		}
	}
}

func TestProcessPageWaitsBetweenRequestsAndReportsStatus(t *testing.T) {
	var agents []string
	srv := sitemapServer(t, &agents)
	src := New(srv.URL, srv.Client())
	src.pause = 50 * time.Millisecond
	ctx := context.Background()

	start := time.Now()
	page, err := src.ProcessPage(ctx, domain.BillKey{Number: 5564, Year: 2025})
	if err != nil || string(page) != `<html><main>página</main></html>` {
		t.Fatalf("página: %q %v", page, err)
	}
	_, err = src.ProcessPage(ctx, domain.BillKey{Number: 1, Year: 2025})

	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("esperava erro de status, veio %v", err)
	}
	if time.Since(start) < 50*time.Millisecond {
		t.Fatal("o segundo pedido deveria esperar a pausa")
	}
}
