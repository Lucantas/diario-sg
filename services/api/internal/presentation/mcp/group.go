package mcp

import (
	"context"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type groupInput struct {
	By            string  `json:"por" jsonschema:"cnpj, processo, orgao ou tipo"`
	IncludePublic bool    `json:"incluir_orgaos_publicos,omitempty" jsonschema:"no agrupamento por cnpj, incluir o Município, fundações, fundos, SG-PREVI e Câmara (padrão: não)"`
	Query         string  `json:"consulta,omitempty" jsonschema:"termos da busca em português; aceita \"frase exata\", OU e -excluir"`
	Diario        string  `json:"diario,omitempty" jsonschema:"diario_prefeitura ou diario_camara; vazio agrupa os dois"`
	Type          string  `json:"tipo,omitempty" jsonschema:"tipo do ato, como em buscar_atos"`
	Organ         string  `json:"orgao,omitempty" jsonschema:"sigla do órgão da Prefeitura, como SEMED"`
	From          string  `json:"de,omitempty" jsonschema:"data inicial da edição, AAAA-MM-DD"`
	To            string  `json:"ate,omitempty" jsonschema:"data final da edição, AAAA-MM-DD"`
	MinValue      float64 `json:"valor_min,omitempty" jsonschema:"só atos que citam ao menos um valor a partir deste, em reais"`
	MaxValue      float64 `json:"valor_max,omitempty" jsonschema:"só atos que citam ao menos um valor até este, em reais"`
	Modality      string  `json:"modalidade,omitempty" jsonschema:"modalidade da contratação, como em buscar_atos"`
	MainMin       float64 `json:"valor_principal_min,omitempty" jsonschema:"só atos cujo valor principal é a partir deste, em reais"`
	MainMax       float64 `json:"valor_principal_max,omitempty" jsonschema:"só atos cujo valor principal é até este, em reais"`
	Limit         int     `json:"limite,omitempty" jsonschema:"grupos, de 1 a 50 (padrão 20)"`
}

type actRefDTO struct {
	GazetteID   string `json:"edicao_id"`
	Position    int    `json:"posicao"`
	Diario      string `json:"diario"`
	PublishedAt string `json:"data"`
	Title       string `json:"titulo"`
}

type groupDTO struct {
	Key           string      `json:"chave"`
	Name          string      `json:"nome,omitempty"`
	Acts          int         `json:"atos"`
	First         string      `json:"primeira_data"`
	Last          string      `json:"ultima_data"`
	MaxValueCents int64       `json:"maior_valor_centavos"`
	Examples      []actRefDTO `json:"exemplos"`
}

type groupOutput struct {
	By          string        `json:"por"`
	MatchedActs int           `json:"atos_encontrados"`
	Groups      []groupDTO    `json:"grupos"`
	Coverage    []coverageDTO `json:"cobertura"`
	Alerts      []string      `json:"alertas_coleta,omitempty"`
}

const groupDescription = "Agrupa os atos encontrados por cnpj, processo, orgao ou tipo, com os mesmos filtros de buscar_atos: " +
	"quantos atos cada grupo tem, a primeira e a última data, o maior valor citado num ato do grupo e até 3 atos de exemplo. " +
	"Os grupos vêm do maior para o menor. Por padrão, o agrupamento por cnpj deixa de fora os CNPJs de órgãos públicos. " +
	"maior_valor_centavos é o maior valor citado, não a soma: um ato cita valores de itens, parcelas e totais. " +
	"Use entidade para ver todos os atos de um CNPJ ou processo."

func (s *server) group(ctx context.Context, _ *sdk.CallToolRequest, in groupInput) (*sdk.CallToolResult, groupOutput, error) {
	f, err := filterOf(searchInput{Query: in.Query, Diario: in.Diario, Type: in.Type, Organ: in.Organ, From: in.From, To: in.To,
		MinValue: in.MinValue, MaxValue: in.MaxValue, Modality: in.Modality, MainMin: in.MainMin, MainMax: in.MainMax})
	if err != nil {
		return nil, groupOutput{}, err
	}
	res, err := s.groupActs.Execute(ctx, domain.GroupQuery{Filter: f, By: domain.GroupBy(in.By), IncludePublic: in.IncludePublic, Limit: in.Limit})
	if err != nil {
		return nil, groupOutput{}, err
	}
	cs, err := s.cachedCoverage(ctx)
	if err != nil {
		return nil, groupOutput{}, err
	}
	out := groupOutput{By: in.By, MatchedActs: res.MatchedActs, Groups: make([]groupDTO, 0, len(res.Groups)),
		Coverage: coveragesOf(cs), Alerts: collectionAlerts(cs)}
	for _, g := range res.Groups {
		out.Groups = append(out.Groups, groupOf(domain.GroupBy(in.By), g))
	}
	return nil, out, nil
}

func groupOf(by domain.GroupBy, g domain.ActGroup) groupDTO {
	dto := groupDTO{Key: g.Key, Name: groupName(by, g.Key), Acts: g.Acts, First: g.First.Format(time.DateOnly),
		Last: g.Last.Format(time.DateOnly), MaxValueCents: g.MaxValueCents, Examples: make([]actRefDTO, 0, len(g.Examples))}
	for _, e := range g.Examples {
		dto.Examples = append(dto.Examples, actRefDTO{GazetteID: e.GazetteID, Position: e.Position,
			Diario: domain.SourceOrDefault(e.Source), PublishedAt: e.PublishedAt.Format(time.DateOnly), Title: e.Title})
	}
	return dto
}

func groupName(by domain.GroupBy, key string) string {
	switch by {
	case domain.GroupByOrgan:
		if key == "" {
			return "sem órgão identificado"
		}
		return domain.OrganName(key)
	case domain.GroupByCNPJ:
		name, _ := domain.PublicBody(key)
		return name
	}
	return ""
}
