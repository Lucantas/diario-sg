package http

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSecurityTxtPointsToPrivateReportsAndStaysValid(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	api := &API{PublicWebURL: "https://diariosg.com.br/", Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	rec := httptest.NewRecorder()

	api.securityTxt(func() time.Time { return now }).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil))

	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("esperava 200 em text/plain, veio %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	for _, line := range []string{
		"Contact: https://github.com/Lucantas/diario-sg/security/advisories/new",
		"Expires: 2027-04-04T12:00:00Z",
		"Canonical: https://diariosg.com.br/.well-known/security.txt",
		"Preferred-Languages: pt, en",
	} {
		if !strings.Contains(body, line+"\n") {
			t.Errorf("faltou %q em:\n%s", line, body)
		}
	}
}

func TestSecurityTxtIsRouted(t *testing.T) {
	api := &API{PublicWebURL: "https://diariosg.com.br", Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	rec := httptest.NewRecorder()

	api.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil))

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Contact: ") {
		t.Fatalf("esperava o security.txt, veio %d %s", rec.Code, rec.Body.String())
	}
}
