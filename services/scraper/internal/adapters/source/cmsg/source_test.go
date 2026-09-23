package cmsg

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

type server struct {
	mu      sync.Mutex
	methods []string
	status  map[string]int
}

func (s *server) handler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.methods = append(s.methods, r.Method+" "+r.URL.Path)
	s.mu.Unlock()
	if r.Header.Get("User-Agent") == "" || !strings.HasPrefix(r.Header.Get("User-Agent"), "diario-sg-bot") {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	code, ok := s.status[r.URL.Path]
	if !ok {
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	if code == http.StatusOK && r.Method == http.MethodGet {
		_, _ = io.WriteString(w, "%PDF-1.3 "+r.URL.Path)
	}
}

func newSource(t *testing.T, status map[string]int) (*Source, *server) {
	t.Helper()
	s := &server{status: status}
	ts := httptest.NewServer(http.HandlerFunc(s.handler))
	t.Cleanup(ts.Close)
	src, err := New(ts.URL + "/diariooficialeletronico/")
	if err != nil {
		t.Fatal(err)
	}
	src.delay = 0
	return src, s
}

func day(d int) time.Time { return time.Date(2025, 11, d, 0, 0, 0, 0, time.UTC) }

func TestListEditionsProbesEachDayWithHead(t *testing.T) {
	src, srv := newSource(t, map[string]int{
		"/diariooficialeletronico/PUBLICACOES/2025-11-03.pdf": http.StatusOK,
		"/diariooficialeletronico/PUBLICACOES/2025-11-05.pdf": http.StatusOK,
	})

	got, err := src.ListEditions(context.Background(), day(3), day(5).Add(15*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || !got[0].PublishedAt.Equal(day(3)) || !got[1].PublishedAt.Equal(day(5)) {
		t.Fatalf("edições: %+v", got)
	}
	if !strings.HasSuffix(got[0].URL, "/diariooficialeletronico/PUBLICACOES/2025-11-03.pdf") || got[0].Source != domain.SourceDiarioCamara {
		t.Errorf("edição inesperada: %+v", got[0])
	}
	for _, m := range srv.methods {
		if !strings.HasPrefix(m, "HEAD ") {
			t.Errorf("listar deveria usar só HEAD: %v", srv.methods)
		}
	}
	if len(srv.methods) != 3 {
		t.Errorf("esperava um pedido por dia, veio %v", srv.methods)
	}
}

func TestListEditionsFailsOnUnexpectedStatus(t *testing.T) {
	src, _ := newSource(t, map[string]int{"/diariooficialeletronico/PUBLICACOES/2025-11-04.pdf": http.StatusInternalServerError})

	_, err := src.ListEditions(context.Background(), day(3), day(5))

	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("esperava erro com o status, veio %v", err)
	}
}

func TestListEditionsStopsWhenCancelled(t *testing.T) {
	src, _ := newSource(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := src.ListEditions(ctx, day(1), day(30)); err == nil {
		t.Fatal("esperava erro de contexto cancelado")
	}
}

func TestDownloadGetsThePDF(t *testing.T) {
	path := "/diariooficialeletronico/PUBLICACOES/2025-11-03.pdf"
	src, _ := newSource(t, map[string]int{path: http.StatusOK})
	editions, err := src.ListEditions(context.Background(), day(3), day(3))
	if err != nil || len(editions) != 1 {
		t.Fatalf("listar: %v %v", editions, err)
	}

	rc, err := src.Download(context.Background(), editions[0])
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	body, _ := io.ReadAll(rc)

	if string(body) != "%PDF-1.3 "+path {
		t.Errorf("corpo inesperado: %q", body)
	}
}

func TestNewRejectsInvalidURLAndNamesTheSource(t *testing.T) {
	if _, err := New("não é url"); err == nil {
		t.Error("URL inválida deveria dar erro")
	}
	src, err := New(DefaultURL)
	if err != nil || src.Name() != domain.SourceDiarioCamara {
		t.Errorf("fonte padrão: %v %v", src, err)
	}
}
