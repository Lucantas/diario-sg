package mcp

import (
	"context"
	"fmt"
	"net/url"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxAgentsPerCall = 20

const agentsDescription = "Agentes políticos de São Gonçalo com a remuneração mês a mês: prefeito, vice, secretários municipais e " +
	"Procurador-Geral pela folha da Prefeitura (valor bruto, de 10/2010 em diante), e vereadores pela folha da Câmara (vencimentos, " +
	"descontos e líquido, de 2017 em diante), com partido e nome parlamentar do SICAM. nome filtra por parte do nome, do nome parlamentar " +
	"ou da lotação; cargo é prefeito, vice_prefeito, secretario, procurador_geral ou vereador. Traz também o subsídio fixado em lei " +
	"(normas com a busca no Diário). Só agentes políticos têm nome aqui; não use para procurar outros servidores. Dezembro da Câmara " +
	"traz o 13º junto, e nome é a única chave entre as folhas. Vêm até 20 agentes por chamada; total diz quantos há e pular " +
	"avança a lista."

type agentsInput struct {
	Name string `json:"nome,omitempty" jsonschema:"parte do nome, do nome parlamentar ou da lotação"`
	Role string `json:"cargo,omitempty" jsonschema:"prefeito, vice_prefeito, secretario, procurador_geral ou vereador"`
	Skip int    `json:"pular,omitempty" jsonschema:"quantos agentes pular do início da lista, para ver os seguintes"`
}

type agentMonthDTO struct {
	Month         string `json:"mes"`
	Office        string `json:"lotacao"`
	GrossCents    int64  `json:"bruto_centavos"`
	DiscountCents *int64 `json:"descontos_centavos,omitempty"`
	NetCents      *int64 `json:"liquido_centavos,omitempty"`
}

type agentDTO struct {
	Body              string          `json:"orgao"`
	Role              string          `json:"cargo"`
	Name              string          `json:"nome"`
	Party             string          `json:"partido,omitempty"`
	ParliamentaryName string          `json:"nome_parlamentar,omitempty"`
	First             string          `json:"primeiro_mes"`
	Last              string          `json:"ultimo_mes"`
	DiarioSearch      string          `json:"busca_no_diario"`
	Months            []agentMonthDTO `json:"meses"`
}

type normDTO struct {
	Role       string `json:"cargo"`
	FromYear   int    `json:"de"`
	ToYear     int    `json:"ate"`
	ValueCents int64  `json:"subsidio_centavos"`
	Norm       string `json:"norma"`
	Diario     string `json:"diario"`
	Search     string `json:"busca_no_diario"`
}

type agentsOutput struct {
	Total    int               `json:"total"`
	Agents   []agentDTO        `json:"agentes"`
	Norms    []normDTO         `json:"subsidio_fixado"`
	Coverage map[string]string `json:"cobertura"`
}

func (s *server) agents(ctx context.Context, _ *sdk.CallToolRequest, in agentsInput) (*sdk.CallToolResult, agentsOutput, error) {
	if in.Skip < 0 {
		return nil, agentsOutput{}, fmt.Errorf("pular não pode ser negativo: %d", in.Skip)
	}
	rep, err := s.politicalAgents.Execute(ctx, in.Role, in.Name)
	if err != nil {
		return nil, agentsOutput{}, err
	}
	page := rep.Agents[min(in.Skip, len(rep.Agents)):]
	page = page[:min(len(page), maxAgentsPerCall)]
	out := agentsOutput{Total: len(rep.Agents), Agents: make([]agentDTO, 0, len(page)), Coverage: map[string]string{}}
	for _, a := range page {
		dto := agentDTO{Body: a.Body, Role: a.Role, Name: a.Name, Party: a.Party, ParliamentaryName: a.ParliamentaryName,
			First: a.FirstMonth().Format("2006-01"), Last: a.LastMonth().Format("2006-01"), DiarioSearch: s.searchURL(`"` + a.Name + `"`),
			Months: make([]agentMonthDTO, 0, len(a.Months))}
		for _, m := range a.Months {
			dto.Months = append(dto.Months, agentMonthDTO{Month: m.Month.Format("2006-01"), Office: m.Office, GrossCents: m.GrossCents,
				DiscountCents: m.DiscountCents, NetCents: m.NetCents})
		}
		out.Agents = append(out.Agents, dto)
	}
	for _, n := range rep.Norms {
		out.Norms = append(out.Norms, normDTO{Role: n.Role, FromYear: n.FromYear, ToYear: n.ToYear, ValueCents: n.Cents, Norm: n.Norm,
			Diario: n.Diario, Search: s.searchURL(n.Search)})
	}
	for _, c := range rep.Coverage {
		out.Coverage[c.Body] = c.From.Format("2006-01") + " a " + c.To.Format("2006-01")
	}
	return nil, out, nil
}

func (s *server) searchURL(query string) string {
	return s.webURL + "/?" + url.Values{"q": {query}}.Encode()
}
