package siconfi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRREOAsksForTheMunicipalityAnnex01(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer srv.Close()
	s := New(srv.URL+"/", srv.Client())
	s.pause = 0

	body, err := s.RREO(context.Background(), 2025, 6)

	if err != nil || string(body) != `{"items":[]}` {
		t.Fatal(string(body), err)
	}
	for _, want := range []string{"an_exercicio=2025", "nr_periodo=6", "id_ente=3304904", "co_tipo_demonstrativo=RREO", "no_anexo=RREO-Anexo+01"} {
		if !strings.Contains(got, want) {
			t.Errorf("consulta sem %s: %s", want, got)
		}
	}
}

func TestRREOFailsOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer srv.Close()
	s := New(srv.URL+"/", srv.Client())
	s.pause = 0

	if _, err := s.RREO(context.Background(), 2025, 6); err == nil || !strings.Contains(err.Error(), "502") {
		t.Errorf("erro HTTP: %v", err)
	}
}
