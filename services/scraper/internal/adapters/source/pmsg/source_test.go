package pmsg

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	editions, pages := parseListing(fixture(t, "listing_page1.html"))
	want := []string{"2026-10-22", "2026-10-21", "2026-10-20", "2026-10-19", "2026-10-16", "2026-10-15", "2026-10-14", "2026-10-13", "2026-10-09", "2026-10-08", "2026-07-14"}
	var got []string
	for _, e := range editions {
		if e.path != "diario/download/pdf/"+e.date.Format("2006_01_02")+".pdf" {
			t.Errorf("caminho inesperado para %s: %s", e.date.Format(time.DateOnly), e.path)
		}
		got = append(got, e.date.Format(time.DateOnly))
	}
	sort.Sort(sort.Reverse(sort.StringSlice(got)))
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("esperava %v, veio %v", want, got)
	}
	sort.Ints(pages)
	if fmt.Sprint(pages) != "[2 3 4]" {
		t.Errorf("página 1 deveria apontar para 2, 3 e 4, veio %v", pages)
	}
}

func TestListEditionsFollowsPaginationAndThrottles(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	var times []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.URL.Query().Get("page"))
		times = append(times, time.Now())
		mu.Unlock()
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent não identificado: %q", r.Header.Get("User-Agent"))
		}
		q := r.URL.Query()
		if r.Method != http.MethodGet || r.URL.Path != "/" || q.Get("start_date") != "2026-09-01" || q.Get("end_date") != "2026-10-22" || !q.Has("q") {
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.RequestURI())
		}
		if q.Get("page") == "1" {
			w.Write([]byte(fixture(t, "listing_page1.html")))
			return
		}
		w.Write([]byte(fixture(t, "listing_page4.html")))
	}))
	defer srv.Close()

	s, err := New(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	s.delay = 30 * time.Millisecond
	from := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)
	editions, err := s.ListEditions(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(editions) != 15 {
		t.Fatalf("esperava 15 edições (10 + 5), veio %d: %+v", len(editions), editions)
	}
	if editions[0].PublishedAt.Format(time.DateOnly) != "2026-09-01" || editions[14].URL != srv.URL+"/diario/download/pdf/2026_10_22.pdf" {
		t.Errorf("ordem ou URL inesperada: %+v", editions)
	}
	if strings.Join(calls, " ") != "1 2 3 4" {
		t.Errorf("esperava visitar as páginas 1 a 4 uma vez cada, veio %v", calls)
	}
	if gap := times[1].Sub(times[0]); gap < s.delay-5*time.Millisecond {
		t.Errorf("pausa entre requisições não respeitada: %v", gap)
	}
}

func TestListEditionsKeepsExtraEditionsOfTheSameDay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fixture(t, "listing_extra.html")))
	}))
	defer srv.Close()
	s, _ := New(srv.URL + "/")
	s.delay = 0

	editions, err := s.ListEditions(context.Background(),
		time.Date(2024, 8, 21, 0, 0, 0, 0, time.UTC), time.Date(2024, 8, 23, 0, 0, 0, 0, time.UTC))

	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range editions {
		got = append(got, e.PublishedAt.Format(time.DateOnly)+" "+strings.TrimPrefix(e.URL, srv.URL))
	}
	want := []string{
		"2024-08-21 /diario/download/pdf/2024_08_21.pdf",
		"2024-08-21 /diario/download/pdf/2024_08_21_1.pdf",
		"2024-08-22 /diario/download/pdf/2024_08_22.pdf",
		"2024-08-23 /diario/download/pdf/2024_08_23.pdf",
		"2024-08-23 /diario/download/pdf/2024_08_23_1.pdf",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("esperava\n%s\nveio\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))
	}
}

func TestExtraEditionsGetTheirOwnStoragePath(t *testing.T) {
	main := editionFor("https://do.pmsg.rj.gov.br", 2024, 8, 21)
	extra := main
	extra.URL = "https://do.pmsg.rj.gov.br/diario/download/pdf/2024_08_21_1.pdf"

	if main.StoragePath() == extra.StoragePath() {
		t.Errorf("edição extra sobrescreveria a principal em %s", main.StoragePath())
	}
}

func TestListEditionsFiltersByPeriod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fixture(t, "listing_page4.html")))
	}))
	defer srv.Close()
	s, _ := New(srv.URL + "/")
	s.delay = 0
	editions, err := s.ListEditions(context.Background(),
		time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 3, 23, 0, 0, 0, time.UTC))
	if err != nil || len(editions) != 2 {
		t.Fatalf("esperava só 02 e 03/09, veio %+v (%v)", editions, err)
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

func TestListEditionsFailsInsteadOfTruncatingSilently(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := 1
		fmt.Sscan(r.URL.Query().Get("page"), &page)
		fmt.Fprintf(w, `<a href="/diario/download/pdf/2026_01_%02d.pdf">x</a><a href="/?start_date=2026-01-01&amp;page=%d">next</a>`, page%28+1, page+1)
	}))
	defer srv.Close()
	s, _ := New(srv.URL + "/")
	s.delay = 0
	_, err := s.ListEditions(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC))
	if err == nil || !strings.Contains(err.Error(), "páginas") {
		t.Fatalf("esperava erro por excesso de páginas, veio %v", err)
	}
}
