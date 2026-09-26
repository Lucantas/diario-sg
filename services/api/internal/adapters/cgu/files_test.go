package cgu

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestZipCSVsReadsEveryFileInLatin1(t *testing.T) {
	body := zipOf(t, map[string]string{
		"A.csv": "\"Município\";\"Valor\"\n\"S\xc3O GON\xc7ALO\";\"1,00\"\n",
		"B.csv": "\"x\"\n\"1\"\n\"2\"\n",
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/emendas-parlamentares/UNICO" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	rows := map[string][][]string{}

	sum, err := unthrottled(New(srv.URL, srv.Client())).ZipCSVs(context.Background(), "emendas-parlamentares/UNICO", func(name string, header, row []string) error {
		rows[name] = append(rows[name], row)
		return nil
	})

	if err != nil || sum == "" || len(rows["B.csv"]) != 2 || rows["A.csv"][0][0] != "SÃO GONÇALO" {
		t.Fatalf("veio %v %v %q", sum, err, rows)
	}
}

func TestLatestMonthReadsTheMonthlyListing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`arquivos.push({"ano" : "2026", "mes" : "08", "dia" : "", "origem" : "Transferencias"})
arquivos.push({"ano" : "2026", "mes" : "09", "dia" : "", "origem" : "Transferencias"})`))
	}))
	defer srv.Close()

	month, err := unthrottled(New(srv.URL, srv.Client())).LatestMonth(context.Background(), "transferencias")

	if err != nil || month.Format("2006-01") != "2026-09" {
		t.Fatalf("veio %v %v", month, err)
	}
}

func unthrottled(s *Source) *Source {
	s.minInterval = 0
	return s
}

func TestHumanVerificationIsReportedAsSuch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("<title>Human Verification</title>"))
	}))
	defer srv.Close()

	_, err := unthrottled(New(srv.URL, srv.Client())).ZipCSVs(context.Background(), "transferencias/202601", func(string, []string, []string) error { return nil })

	if !errors.Is(err, ErrHumanVerification) {
		t.Fatalf("esperava verificação humana, veio %v", err)
	}
}

func TestDownloadsAreSpacedOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`arquivos.push({"ano" : "2026", "mes" : "09"})`))
	}))
	defer srv.Close()
	s := New(srv.URL, srv.Client())
	s.minInterval = 50 * time.Millisecond
	start := time.Now()

	for range 3 {
		if _, err := s.LatestMonth(context.Background(), "transferencias"); err != nil {
			t.Fatal(err)
		}
	}

	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("três pedidos em %v", elapsed)
	}
}
