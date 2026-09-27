package mcp

import (
	"context"
	"net/url"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const normsDescription = "Leis, leis complementares, a Lei Orgânica e decretos de São Gonçalo, da consulta de leis da Prefeitura (SIAPEGOV). " +
	"Com tipo (lei, lei_complementar, lei_organica ou decreto) e numero (como 1406/2022), traz aquela norma; com texto, busca na ementa e no autor " +
	"(até 50, as mais recentes primeiro; tipo opcional). Cada norma traz a ementa, o autor quando a consulta informa (em leis de vereador, o projeto de lei), " +
	"a data de promulgação, o link do texto integral e busca_diario, a busca pronta para achar no Diário os atos que citam o número " +
	"(use com buscar_atos). A consulta não diz o que cada norma alterou."

type normsInput struct {
	Kind   string `json:"tipo,omitempty" jsonschema:"lei, lei_complementar, lei_organica ou decreto"`
	Number string `json:"numero,omitempty" jsonschema:"número/ano, como 1406/2022"`
	Text   string `json:"texto,omitempty" jsonschema:"palavras da ementa ou do autor"`
}

type normOutDTO struct {
	Tipo         string `json:"tipo"`
	Numero       string `json:"numero"`
	Autor        string `json:"autor,omitempty"`
	Ementa       string `json:"ementa"`
	Promulgacao  string `json:"promulgacao,omitempty"`
	TextoIntegro string `json:"texto_integral,omitempty"`
	BuscaDiario  string `json:"busca_diario"`
	BuscaSite    string `json:"busca_no_site"`
}

type normsOutput struct {
	Normas []normOutDTO `json:"normas"`
}

func (s *server) norms(ctx context.Context, _ *sdk.CallToolRequest, in normsInput) (*sdk.CallToolResult, normsOutput, error) {
	norms, err := s.findNorms.Execute(ctx, in.Kind, in.Number, in.Text)
	if err != nil {
		return nil, normsOutput{}, err
	}
	out := normsOutput{Normas: make([]normOutDTO, len(norms))}
	for i, n := range norms {
		search := domain.NormDiarioSearch(n)
		out.Normas[i] = normOutDTO{Tipo: string(n.Kind), Numero: n.Label(), Autor: n.Author, Ementa: n.Summary, TextoIntegro: n.TextURL,
			BuscaDiario: search, BuscaSite: s.webURL + "/?" + url.Values{"q": {search}}.Encode()}
		if n.PromulgatedOn != nil {
			out.Normas[i].Promulgacao = n.PromulgatedOn.Format(time.DateOnly)
		}
	}
	return nil, out, nil
}
