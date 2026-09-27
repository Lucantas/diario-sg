package pmsgportal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommitmentsAsksForTheWholeYearOfAnEntity(t *testing.T) {
	var got, agent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, agent = r.URL.Path+"?"+r.URL.RawQuery, r.UserAgent()
		_, _ = w.Write([]byte(`{"empenhos":[]}`))
	}))
	defer srv.Close()
	s := New(srv.URL+"/api/", srv.Client())
	s.pause = 0

	body, err := s.Commitments(context.Background(), 2025, 18)

	if err != nil || string(body) != `{"empenhos":[]}` {
		t.Fatal(string(body), err)
	}
	if !strings.HasPrefix(got, "/api/execucao/empenhos/empenhos?") || !strings.Contains(got, "ano=2025") || !strings.Contains(got, "id_entidade=18") ||
		!strings.Contains(got, "nome_razao=&") {
		t.Errorf("consulta: %s", got)
	}
	if agent != userAgent {
		t.Errorf("User-Agent: %q", agent)
	}
}

func TestEntitiesFailsOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer srv.Close()
	s := New(srv.URL+"/", srv.Client())
	s.pause = 0

	if _, err := s.Entities(context.Background()); err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("erro: %v", err)
	}
}
