package pncp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func contractJSON(seq int, kind string) string {
	return fmt.Sprintf(`{"numeroControlePNCP":"28636579000100-2-%06d/2025","anoContrato":2025,"sequencialContrato":%d,"processo":"24.501/2024",`+
		`"numeroContratoEmpenho":"SEMAD 004","tipoPessoa":"%s","niFornecedor":"15106169000106","nomeRazaoSocialFornecedor":"FSM LTDA",`+
		`"objetoContrato":"Telefonia","valorGlobal":14880.5,"dataAssinatura":"2024-12-23","dataPublicacaoPncp":"2025-01-09T16:22:39",`+
		`"dataVigenciaInicio":"2025-01-01","dataVigenciaFim":"2029-12-31","tipoContrato":{"nome":"Contrato (termo inicial)"},`+
		`"orgaoEntidade":{"cnpj":"28636579000100"},"unidadeOrgao":{"nomeUnidade":"PREFEITURA"}}`, seq, seq, kind)
}

func newPNCP(t *testing.T, h http.HandlerFunc) *Source {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	s := New(srv.URL+"/api/consulta/v1/", srv.Client())
	s.pause, s.limitWait = 0, time.Millisecond
	return s
}

func TestContractsFollowsThePages(t *testing.T) {
	var queries []string
	src := newPNCP(t, func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		if r.URL.Query().Get("pagina") == "1" {
			fmt.Fprintf(w, `{"data":[%s],"paginasRestantes":1}`, contractJSON(1, "PJ"))
			return
		}
		fmt.Fprintf(w, `{"data":[%s],"paginasRestantes":0}`, contractJSON(2, "PF"))
	})
	var got []domain.PNCPContract
	var companies []bool

	_, err := src.Contracts(context.Background(), "28636579000100", 2025, func(c domain.PNCPContract, company bool) error {
		got, companies = append(got, c), append(companies, company)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !companies[0] || companies[1] || len(queries) != 2 || !strings.Contains(queries[0], "dataInicial=20250101") {
		t.Fatalf("contratos %+v, empresas %v, consultas %v", got, companies, queries)
	}
	c := got[0]
	if c.Sequence != 1 || c.ValueCents != 1488050 || c.SignedAt.Format(time.DateOnly) != "2024-12-23" || c.PublishedAt.Format(time.DateOnly) != "2025-01-09" ||
		c.Process != "24.501/2024" || c.ProcessKey() != "245012024" || c.URL() != "https://pncp.gov.br/app/contratos/28636579000100/2025/1" {
		t.Errorf("contrato: %+v", c)
	}
}

func TestContractsOfAnEmptyYear(t *testing.T) {
	src := newPNCP(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	n := 0

	if _, err := src.Contracts(context.Background(), "28636579000100", 2021, func(domain.PNCPContract, bool) error { n++; return nil }); err != nil || n != 0 {
		t.Fatalf("ano vazio: %d %v", n, err)
	}
}

func TestContractsWaitsWhenTheRateLimitPageComesBack(t *testing.T) {
	var calls atomic.Int32
	src := newPNCP(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			_, _ = w.Write([]byte("<html><title>Limite de Requisições Excedido</title></html>"))
			return
		}
		fmt.Fprintf(w, `{"data":[%s],"paginasRestantes":0}`, contractJSON(1, "PJ"))
	})
	n := 0

	if _, err := src.Contracts(context.Background(), "28636579000100", 2025, func(domain.PNCPContract, bool) error { n++; return nil }); err != nil || n != 1 {
		t.Fatalf("depois do limite: %d %v", n, err)
	}
}
