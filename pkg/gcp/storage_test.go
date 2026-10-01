package gcp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type noToken struct{}

func (noToken) Token(context.Context) (string, error) { return "", nil }

func TestGetTellsAMissingObjectApartFromOtherFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "existe"):
			io.WriteString(w, "conteúdo")
		case strings.Contains(r.URL.Path, "falta"):
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	s := NewStorage("b", strings.TrimPrefix(srv.URL, "http://"), noToken{})

	rc, err := s.Get(context.Background(), "ocr/existe.json")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(rc)
	rc.Close()
	_, missing := s.Get(context.Background(), "ocr/falta.json")
	_, broken := s.Get(context.Background(), "ocr/quebrado.json")

	if string(body) != "conteúdo" {
		t.Errorf("corpo: %q", body)
	}
	if !errors.Is(missing, ErrObjectNotFound) {
		t.Errorf("404 deveria ser ErrObjectNotFound: %v", missing)
	}
	if broken == nil || errors.Is(broken, ErrObjectNotFound) {
		t.Errorf("500 não é objeto ausente: %v", broken)
	}
}
