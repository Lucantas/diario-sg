//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func agentMonth(y int, m time.Month) time.Time { return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC) }

func TestPoliticalAgentsReplaceTheirMonthsAndShowInAPIAndMCP(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)
	repo := postgres.NewPoliticalAgentRepo(db)
	ctx := context.Background()
	discount, net := int64(581374), int64(1602669)
	vereador := domain.AgentPay{Body: domain.BodyCamara, Month: agentMonth(2026, 7), Name: "CLAUDIO LUIZ ABREU DA SILVA", NameKey: "CLAUDIO LUIZ ABREU DA SILVA",
		Role: domain.RoleVereador, Office: "VEREADOR CACAU", GrossCents: 2184043, DiscountCents: &discount, NetCents: &net}
	prefeito := domain.AgentPay{Body: domain.BodyPrefeitura, Month: agentMonth(2026, 7), Name: "NELSON RUAS DOS SANTOS", NameKey: "NELSON RUAS DOS SANTOS",
		Role: domain.RolePrefeito, Office: "GABINETE DO PREFEITO", GrossCents: 2381322}
	august := prefeito
	august.Month, august.GrossCents = agentMonth(2026, 8), 1
	cacau := domain.Councillor{Legislature: 4, Name: "CLAUDIO LUÍS ABREU DA SILVA", NameKey: "CLAUDIO LUIS ABREU DA SILVA", ParliamentaryName: "CACAU", Party: "MDB", Situation: "Ativo"}
	if err := repo.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	if err := repo.SavePoliticalAgents(ctx, domain.PoliticalAgentLoad{Body: domain.BodyCamara, From: agentMonth(2026, 7), To: agentMonth(2026, 8),
		Pay: []domain.AgentPay{vereador}, Councillors: []domain.Councillor{cacau}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SavePoliticalAgents(ctx, domain.PoliticalAgentLoad{Body: domain.BodyPrefeitura, From: agentMonth(2026, 7), To: agentMonth(2026, 8),
		Pay: []domain.AgentPay{prefeito, august}}); err != nil {
		t.Fatal(err)
	}
	august.GrossCents = 2381322
	if err := repo.SavePoliticalAgents(ctx, domain.PoliticalAgentLoad{Body: domain.BodyPrefeitura, From: agentMonth(2026, 7), To: agentMonth(2026, 8),
		Pay: []domain.AgentPay{prefeito, august}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SavePoliticalAgents(ctx, domain.PoliticalAgentLoad{Body: domain.BodyPrefeitura, From: agentMonth(2026, 8), To: agentMonth(2026, 8),
		Pay: []domain.AgentPay{vereador}}); err == nil {
		t.Error("linha da Câmara aceita numa carga da Prefeitura")
	}

	var res struct {
		Agents []struct {
			Role   string `json:"role"`
			Name   string `json:"name"`
			Party  string `json:"party"`
			Months []struct {
				Month      string `json:"month"`
				GrossCents int64  `json:"gross_cents"`
				NetCents   *int64 `json:"net_cents"`
			} `json:"months"`
		} `json:"agents"`
		Norms    []struct{} `json:"norms"`
		Coverage []struct {
			Body string `json:"body"`
			To   string `json:"to"`
		} `json:"coverage"`
	}
	getJSON(t, srv.URL+"/v1/agentes", &res)
	if len(res.Agents) != 2 || res.Agents[0].Role != domain.RolePrefeito || len(res.Agents[0].Months) != 2 || res.Agents[0].Months[1].GrossCents != 2381322 {
		t.Fatalf("agentes: %+v", res)
	}
	if res.Agents[1].Party != "MDB" || res.Agents[1].Months[0].NetCents == nil || *res.Agents[1].Months[0].NetCents != net || len(res.Norms) != len(domain.SubsidyNorms) {
		t.Errorf("vereador: %+v", res.Agents[1])
	}
	getJSON(t, srv.URL+"/v1/agentes?role=vereador&q=cacau", &res)
	if len(res.Agents) != 1 || res.Agents[0].Name != "CLAUDIO LUIZ ABREU DA SILVA" {
		t.Errorf("filtro: %+v", res.Agents)
	}

	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	out, _ := call[struct {
		Total  int `json:"total"`
		Agents []struct {
			Name   string `json:"nome"`
			Search string `json:"busca_no_diario"`
		} `json:"agentes"`
	}](t, session, "agentes_politicos", map[string]any{"cargo": "prefeito"})
	if out.Total != 1 || out.Agents[0].Name != "NELSON RUAS DOS SANTOS" || out.Agents[0].Search != "https://web.exemplo/?q=%22NELSON+RUAS+DOS+SANTOS%22" {
		t.Errorf("MCP: %+v", out)
	}
	rest, _ := call[struct {
		Total  int `json:"total"`
		Agents []struct {
			Name string `json:"nome"`
		} `json:"agentes"`
	}](t, session, "agentes_politicos", map[string]any{"pular": 1})
	if rest.Total != 2 || len(rest.Agents) != 1 || rest.Agents[0].Name != "CLAUDIO LUIZ ABREU DA SILVA" {
		t.Errorf("MCP pulando o primeiro: %+v", rest)
	}
}
