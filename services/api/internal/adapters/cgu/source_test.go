package cgu

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func latin1(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		out = append(out, byte(r))
	}
	return out
}

func zipped(t *testing.T, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("20260925_CEIS.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const page = `<script>
arquivos.push({"ano" : "2026", "mes" : "09", "dia" : "24", "origem" :  "CEIS"});
arquivos.push({"ano" : "2026", "mes" : "09", "dia" : "25", "origem" :  "CEIS"});
</script>`

func newPortal(t *testing.T, zipBody []byte) *Source {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/download-de-dados/ceis", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(page)) })
	mux.HandleFunc("/download-de-dados/ceis/20260925", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/saida/ceis/20260925_CEIS.zip", http.StatusFound)
	})
	mux.HandleFunc("/saida/ceis/20260925_CEIS.zip", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(zipBody) })
	mux.HandleFunc("/download-de-dados/cnep", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html></html>")) })
	mux.HandleFunc("/download-de-dados/ceis/20260924", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return unthrottled(New(srv.URL+"/download-de-dados/", srv.Client()))
}

func TestLatestDayIsTheNewestPublishedFile(t *testing.T) {
	day, err := newPortal(t, nil).LatestDay(context.Background(), "CEIS")

	if err != nil || !day.Equal(time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("veio %v (%v)", day, err)
	}
}

func TestLatestDayWithoutFilesIsAnError(t *testing.T) {
	if _, err := newPortal(t, nil).LatestDay(context.Background(), "CNEP"); err == nil {
		t.Fatal("página sem arquivo deveria dar erro")
	}
}

func TestRowsFollowsTheRedirectAndDecodesLatin1WithHeader(t *testing.T) {
	csv := latin1("\"CADASTRO\";\"NOME\"\n\"CEIS\";\"CONSTRUÇÃO SÃO GONÇALO\"\n\"CEIS\";\"OUTRA\"\n")
	body := zipped(t, csv)
	var header []string
	var rows [][]string

	sum, err := newPortal(t, body).Rows(context.Background(), "CEIS", time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), func(h, r []string) error {
		header, rows = h, append(rows, r)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(header, "|") != "CADASTRO|NOME" || len(rows) != 2 || rows[0][1] != "CONSTRUÇÃO SÃO GONÇALO" {
		t.Fatalf("cabeçalho %q, linhas %q", header, rows)
	}
	want := sha256.Sum256(body)
	if sum != hex.EncodeToString(want[:]) {
		t.Errorf("sha do zip: %s", sum)
	}
}

func TestRowsOfAnUnavailableDayNamesTheRegisterAndTheDay(t *testing.T) {
	_, err := newPortal(t, nil).Rows(context.Background(), "CEIS", time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), func(_, _ []string) error { return nil })

	if err == nil || !strings.Contains(err.Error(), "CEIS de 2026-09-24") || !strings.Contains(err.Error(), "403") {
		t.Fatalf("erro: %v", err)
	}
}
