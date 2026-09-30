//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
)

func partnerAgentFindings(t *testing.T, url string) []patternFinding {
	t.Helper()
	var res patternsResponse
	getJSON(t, url+"/v1/patterns", &res)
	for _, item := range res.Items {
		if item.ID == "socio_com_nome_de_agente_publico" {
			return item.Findings
		}
	}
	t.Fatalf("padrão ausente: %+v", res.Items)
	return nil
}

func TestPartnerWithTheNameOfAnAppointedPersonOrAPoliticalAgent(t *testing.T) {
	srv, db := newServerFor(t, semedHomologacao2025)
	ctx := context.Background()
	indexAt(t, db, time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC),
		"PORTARIA Nº 7/2025\nO PREFEITO resolve NOMEAR Mariana Barbosa de Souza para Assessora da Secretaria de Obras.")
	loadRegistry(t, postgres.NewRegistryRepo(db), postgres.NewFetchRunRepo(db),
		registryShare{month: "2026-09", name: "F.P. VIEIRA ENGENHARIA LTDA", partner: "MARIANA BARBOSA DE SOUZA"})
	if err := postgres.RefreshPartnerAppointments(ctx, db); err != nil {
		t.Fatal(err)
	}

	appointed := partnerAgentFindings(t, srv.URL)

	if len(appointed) != 1 || appointed[0].Title != "F.P. VIEIRA ENGENHARIA LTDA (14.180.324/0001-63): sócio com o nome de pessoa nomeada ou exonerada no Diário" ||
		len(appointed[0].Acts) != 1 || !strings.Contains(appointed[0].Acts[0].Title, "PORTARIA Nº 7/2025") {
		t.Fatalf("sócio com nome em nomeação: %+v", appointed)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO political_agent_pay (body, month, name, name_key, role, office, gross_cents)
		VALUES ('prefeitura', '2025-01-01', 'Mariana Barbosa de Souza', 'MARIANA BARBOSA DE SOUZA', 'secretario', 'SEMOBI', 100)`); err != nil {
		t.Fatal(err)
	}
	agent := partnerAgentFindings(t, srv.URL)

	if len(agent) != 1 || agent[0].Title != "F.P. VIEIRA ENGENHARIA LTDA (14.180.324/0001-63): sócio com o nome de Mariana Barbosa de Souza, Secretário municipal (SEMOBI)" {
		t.Fatalf("sócio com nome de agente político: %+v", agent)
	}
}
