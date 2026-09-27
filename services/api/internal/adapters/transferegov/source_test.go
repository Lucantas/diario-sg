package transferegov

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestSource(h http.HandlerFunc) (*Source, func()) {
	srv := httptest.NewServer(h)
	s := New(srv.URL+"/", srv.Client())
	s.pause = 0
	return s, srv.Close
}

func TestRowsSendsTheFilterWithAPageLimit(t *testing.T) {
	var got string
	s, done := newTestSource(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path + "?" + r.URL.RawQuery
		_, _ = w.Write([]byte(`[{"id_plano_acao":1}]`))
	})
	defer done()

	body, err := s.Rows(context.Background(), "executor_especial", url.Values{"id_plano_acao": {"in.(1,2)"}})

	if err != nil || string(body) != `[{"id_plano_acao":1}]` {
		t.Fatal(string(body), err)
	}
	if got != "/executor_especial?id_plano_acao=in.%281%2C2%29&limit=1000" {
		t.Errorf("consulta: %s", got)
	}
}

func TestRowsRejectsErrorsAndCutResponses(t *testing.T) {
	full := "[" + strings.TrimSuffix(strings.Repeat(`{},`, pageLimit), ",") + "]"
	for name, h := range map[string]http.HandlerFunc{
		"status":    func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) },
		"postgrest": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"code":"42703"}`)) },
		"cortada":   func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(full)) },
	} {
		s, done := newTestSource(h)
		if _, err := s.Rows(context.Background(), "plano_acao_especial", nil); err == nil {
			t.Errorf("%s: aceito", name)
		}
		done()
	}
}
