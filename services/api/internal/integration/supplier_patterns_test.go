//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
)

func TestSupplierPatternsCrossTheContractWithTheRegistry(t *testing.T) {
	srv, db := newServerFor(t, semedHomologacao2025)
	loadRegistry(t, postgres.NewRegistryRepo(db), postgres.NewFetchRunRepo(db), registryShare{month: "2026-09", name: "F.P. VIEIRA ENGENHARIA LTDA"})

	var res patternsResponse
	getJSON(t, srv.URL+"/v1/patterns", &res)

	byID := map[string]int{}
	for i, item := range res.Items {
		byID[item.ID] = i
	}
	for _, id := range []string{"empresa_nova_contratada", "capital_menor_que_contrato", "socio_em_comum", "endereco_em_comum", "sancionado_contratado"} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("padrão %s ausente: %+v", id, res.Items)
		}
	}
	capital := res.Items[byID["capital_menor_que_contrato"]].Findings
	if len(capital) != 1 || !strings.HasPrefix(capital[0].Title, "F.P. VIEIRA ENGENHARIA LTDA (14.180.324/0001-63): contratação de") ||
		!strings.Contains(capital[0].Title, "com capital social de R$ 500.000,00") || len(capital[0].Acts) != 1 {
		t.Errorf("capital menor que o contrato: %+v", capital)
	}
	if n := len(res.Items[byID["empresa_nova_contratada"]].Findings); n != 0 {
		t.Errorf("empresa aberta em 2011 não é nova: %d", n)
	}
}
