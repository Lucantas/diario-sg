package pmsg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseListingRealHTML(t *testing.T) {
	dates, pages := parseListing(fixture(t, "listing_page1.html"))
	want := []string{"2026-09-18", "2026-09-17", "2026-09-16", "2026-09-15", "2026-09-14"}
	if len(dates) != len(want) {
		t.Fatalf("esperava %d datas, veio %d: %v", len(want), len(dates), dates)
	}
	for i, w := range want {
		if dates[i].Format(time.DateOnly) != w {
			t.Errorf("data %d: esperava %s, veio %s", i, w, dates[i].Format(time.DateOnly))
		}
	}
	if len(pages) != 1 || pages[0] != 2 {
		t.Errorf("página 1 deveria apontar só para a 2, veio %v", pages)
	}

	dates, pages = parseListing(fixture(t, "listing_page2.html"))
	if len(dates) != 4 || dates[3].Format(time.DateOnly) != "2026-09-08" {
		t.Errorf("página 2 inesperada: %v", dates)
	}
	if len(pages) != 1 || pages[0] != 1 {
		t.Errorf("página 2 deveria apontar só para a 1, veio %v", pages)
	}
}

// Servidor que imita o site: POST index devolve a página 1; GET
// index?NumeroPagina=2 devolve a página 2. Registra a ordem e o horário das
// requisições para verificar a pausa entre elas.
func TestListEditionsFollowsPaginationAndThrottles(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	var times []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		times = append(times, time.Now())
		mu.Unlock()
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent não identificado: %q", r.Header.Get("User-Agent"))
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/index":
			r.ParseForm()
			if r.PostForm.Get("Termo") != listingTerm || r.PostForm.Get("DataInicial") != "2026-09-08" || r.PostForm.Get("DataFinal") != "2026-09-21" {
				t.Errorf("formulário inesperado: %v", r.PostForm)
			}
			w.Write([]byte(fixture(t, "listing_page1.html")))
		case r.Method == http.MethodGet && r.URL.Query().Get("NumeroPagina") == "2":
			w.Write([]byte(fixture(t, "listing_page2.html")))
		default:
			http.Error(w, "inesperado", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	s, err := New(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	s.delay = 30 * time.Millisecond
	from := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	editions, err := s.ListEditions(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(editions) != 9 {
		t.Fatalf("esperava 9 edições (5 + 4), veio %d: %+v", len(editions), editions)
	}
	if editions[0].PublishedAt.Format(time.DateOnly) != "2026-09-08" || editions[8].URL != srv.URL+"/diario/2026_09_18.pdf" {
		t.Errorf("ordem ou URL inesperada: %+v", editions)
	}
	if len(calls) != 2 || calls[0] != "POST /index" {
		t.Errorf("esperava POST da página 1 e GET da página 2, veio %v", calls)
	}
	if gap := times[1].Sub(times[0]); gap < s.delay-5*time.Millisecond {
		t.Errorf("pausa entre requisições não respeitada: %v", gap)
	}
}

func TestListEditionsFiltersByPeriod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("NumeroPagina") == "2" {
			w.Write([]byte(fixture(t, "listing_page2.html")))
			return
		}
		w.Write([]byte(fixture(t, "listing_page1.html")))
	}))
	defer srv.Close()
	s, _ := New(srv.URL + "/")
	s.delay = 0
	editions, err := s.ListEditions(context.Background(),
		time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 17, 23, 0, 0, 0, time.UTC))
	if err != nil || len(editions) != 2 {
		t.Fatalf("esperava só 16 e 17/09, veio %+v (%v)", editions, err)
	}
}

func TestDownloadRejectsMissingEdition(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sem edição", http.StatusInternalServerError)
	}))
	defer srv.Close()
	s, _ := New(srv.URL + "/")
	s.delay = 0
	if _, err := s.Download(context.Background(), editionFor(srv.URL, 2026, 9, 20)); err == nil {
		t.Error("status 500 (dia sem edição) deve virar erro")
	}
}

func TestNewRejectsURLWithoutHost(t *testing.T) {
	if _, err := New("diario"); err == nil {
		t.Error("esperava erro para URL sem host")
	}
}
