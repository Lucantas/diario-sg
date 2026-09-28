package mcp

import (
	"context"
	"net/url"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const normsDescription = "Leis, leis complementares, a Lei Orgânica e decretos de São Gonçalo, da consulta de leis da Prefeitura (SIAPEGOV). " +
	"Com tipo (lei, lei_complementar, lei_organica ou decreto) e numero (como 1406/2022), traz aquela norma; com texto, busca na ementa e no autor " +
	"(até 50, as mais recentes primeiro; tipo opcional). Cada norma traz a ementa, o autor quando a consulta informa (em leis de vereador, o projeto de lei), " +
	"a data de promulgação, o link do texto integral e busca_diario, a busca pronta para achar no Diário os atos que citam o número " +
	"(use com buscar_atos). Com tema (meio_ambiente), lista as normas do tema pela ementa, com ou sem texto; regra_tema diz a regra. " +
	"projeto traz a proposição da Câmara que deu origem à lei (ferramenta proposicoes), com a certeza da ligação. " +
	"A consulta não diz o que cada norma alterou."

type normsInput struct {
	Kind   string `json:"tipo,omitempty" jsonschema:"lei, lei_complementar, lei_organica ou decreto"`
	Number string `json:"numero,omitempty" jsonschema:"número/ano, como 1406/2022"`
	Text   string `json:"texto,omitempty" jsonschema:"palavras da ementa ou do autor"`
	Theme  string `json:"tema,omitempty" jsonschema:"meio_ambiente"`
}

type normBillOut struct {
	Processo  string `json:"processo"`
	Documento string `json:"documento"`
	Fase      string `json:"fase"`
	Link      string `json:"link"`
	Certeza   string `json:"certeza"`
}

type normOutDTO struct {
	Tipo         string       `json:"tipo"`
	Numero       string       `json:"numero"`
	Autor        string       `json:"autor,omitempty"`
	Ementa       string       `json:"ementa"`
	Promulgacao  string       `json:"promulgacao,omitempty"`
	TextoIntegro string       `json:"texto_integral,omitempty"`
	BuscaDiario  string       `json:"busca_diario"`
	BuscaSite    string       `json:"busca_no_site"`
	Projeto      *normBillOut `json:"projeto,omitempty"`
}

type normsOutput struct {
	Normas    []normOutDTO `json:"normas"`
	RegraTema string       `json:"regra_tema,omitempty"`
}

func (s *server) norms(ctx context.Context, _ *sdk.CallToolRequest, in normsInput) (*sdk.CallToolResult, normsOutput, error) {
	norms, err := s.findNorms.Execute(ctx, usecase.NormQuery{Kind: in.Kind, Number: in.Number, Text: in.Text, Theme: in.Theme})
	if err != nil {
		return nil, normsOutput{}, err
	}
	out := normsOutput{Normas: make([]normOutDTO, len(norms))}
	if theme, err := domain.ParseTheme(in.Theme); err == nil {
		out.RegraTema = theme.Rule
	}
	for i, n := range norms {
		search := domain.NormDiarioSearch(n)
		out.Normas[i] = normOutDTO{Tipo: string(n.Kind), Numero: n.Label(), Autor: n.Author, Ementa: n.Summary, TextoIntegro: n.TextURL,
			BuscaDiario: search, BuscaSite: s.webURL + "/?" + url.Values{"q": {search}}.Encode()}
		if n.PromulgatedOn != nil {
			out.Normas[i].Promulgacao = n.PromulgatedOn.Format(time.DateOnly)
		}
		if out.Normas[i].Projeto, err = s.originBill(ctx, n); err != nil {
			return nil, normsOutput{}, err
		}
	}
	return nil, out, nil
}

func (s *server) originBill(ctx context.Context, n domain.Norm) (*normBillOut, error) {
	if s.findBills == nil {
		return nil, nil
	}
	bill, certainty, err := s.findBills.ForNorm(ctx, n)
	if err != nil || bill == nil {
		return nil, err
	}
	return &normBillOut{Processo: bill.Key.String(), Documento: bill.DocLabel, Fase: string(domain.BillPhaseOf(*bill)), Link: bill.URL,
		Certeza: string(certainty)}, nil
}
