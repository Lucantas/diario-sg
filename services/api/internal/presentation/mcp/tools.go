package mcp

import (
	"context"
	"errors"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const (
	defaultSearchLimit = 10
	maxActsPerCall     = 20
)

var errInternal = errors.New("erro interno; tente de novo em instantes")

type searchInput struct {
	Query    string  `json:"consulta,omitempty" jsonschema:"termos da busca em português; aceita \"frase exata\", OU e -excluir"`
	Type     string  `json:"tipo,omitempty" jsonschema:"tipo do ato: nomeacao, exoneracao, contrato, aditivo, licitacao, dispensa, decreto, lei, portaria, resolucao, despacho, edital, ata, corrigenda, prestacao_contas ou outro"`
	Organ    string  `json:"orgao,omitempty" jsonschema:"sigla do órgão, como SEMED"`
	From     string  `json:"de,omitempty" jsonschema:"data inicial da edição, AAAA-MM-DD"`
	To       string  `json:"ate,omitempty" jsonschema:"data final da edição, AAAA-MM-DD"`
	MinValue float64 `json:"valor_min,omitempty" jsonschema:"só atos que citam ao menos um valor a partir deste, em reais"`
	MaxValue float64 `json:"valor_max,omitempty" jsonschema:"só atos que citam ao menos um valor até este, em reais"`
	Limit    int     `json:"limite,omitempty" jsonschema:"atos por página, de 1 a 20 (padrão 10)"`
	Offset   int     `json:"deslocamento,omitempty" jsonschema:"quantos atos pular, para paginar"`
}

type searchOutput struct {
	Total    int             `json:"total"`
	Limit    int             `json:"limite"`
	Offset   int             `json:"deslocamento"`
	Acts     []actSummaryDTO `json:"atos"`
	Coverage coverageDTO     `json:"cobertura"`
}

type readInput struct {
	GazetteID string `json:"edicao_id" jsonschema:"edicao_id do ato, como veio de buscar_atos"`
	Position  int    `json:"posicao" jsonschema:"posicao do ato na edição, como veio de buscar_atos"`
}

type readOutput struct {
	GazetteID     string      `json:"edicao_id"`
	Position      int         `json:"posicao"`
	EditionNumber string      `json:"edicao"`
	PublishedAt   string      `json:"data"`
	IsExtra       bool        `json:"extra"`
	Type          string      `json:"tipo"`
	Organ         string      `json:"orgao"`
	OrganName     string      `json:"orgao_nome"`
	Title         string      `json:"titulo"`
	Text          string      `json:"texto"`
	Pages         string      `json:"paginas"`
	Warnings      []string    `json:"avisos"`
	Citation      string      `json:"citacao"`
	Sources       []sourceDTO `json:"fontes"`
	Coverage      coverageDTO `json:"cobertura"`
}

type entityInput struct {
	Kind   string `json:"tipo,omitempty" jsonschema:"cnpj (padrão), processo ou contrato"`
	Number string `json:"numero" jsonschema:"o CNPJ, o número do processo (como 06.10981/2025-7 ou 8421/2023) ou do contrato (como 55/2026 ou 30/FMS/2011)"`
}

type entityOutput struct {
	Kind        string          `json:"tipo"`
	Key         string          `json:"chave"`
	Certainty   string          `json:"certeza,omitempty"`
	Warning     string          `json:"aviso,omitempty"`
	TotalActs   int             `json:"total_atos"`
	TotalCents  int64           `json:"soma_valores_centavos"`
	CountByType map[string]int  `json:"atos_por_tipo"`
	Acts        []actSummaryDTO `json:"atos_recentes"`
	Coverage    coverageDTO     `json:"cobertura"`
}

type sourcesInput struct{}

type lastRunDTO struct {
	FinishedAt string `json:"em"`
	From       string `json:"periodo_de"`
	To         string `json:"periodo_ate"`
	Found      int    `json:"encontradas"`
	Stored     int    `json:"gravadas"`
	Failed     int    `json:"falhas"`
	Error      string `json:"erro,omitempty"`
}

type sourceCoverageDTO struct {
	Name          string      `json:"nome"`
	URL           string      `json:"url"`
	From          string      `json:"de"`
	To            string      `json:"ate"`
	Gazettes      int         `json:"edicoes"`
	Acts          int         `json:"atos"`
	LastCollected string      `json:"ultima_coleta"`
	LastRun       *lastRunDTO `json:"ultima_coleta_tentada,omitempty"`
	Gaps          []string    `json:"lacunas"`
}

type sourcesOutput struct {
	Sources []sourceCoverageDTO `json:"fontes"`
}

func (s *server) register(srv *sdk.Server) {
	readOnly := &sdk.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	sdk.AddTool(srv, &sdk.Tool{Name: "buscar_atos", Annotations: readOnly, Description: "Busca atos no Diário Oficial de São Gonçalo " +
		"(nomeações, contratos, licitações, dispensas, decretos…). Os termos encontrados vêm entre ⟦ ⟧ no trecho. " +
		"Use ler_ato com edicao_id e posicao para o texto completo. Resultado paginado, até 20 atos por chamada."},
		recorded(s, "buscar_atos", s.search))
	sdk.AddTool(srv, &sdk.Tool{Name: "ler_ato", Annotations: readOnly, Description: "Texto completo de um ato, " +
		"com citação pronta (ABNT), link da edição oficial na página do ato e cópia arquivada com SHA-256."},
		recorded(s, "ler_ato", s.read))
	sdk.AddTool(srv, &sdk.Tool{Name: "entidade", Annotations: readOnly, Description: "Atos do Diário que citam um CNPJ, " +
		"um processo ou um contrato: total de atos, contagem por tipo, soma dos valores citados nesses atos e os 20 mais recentes. " +
		"A soma é do que aparece no texto dos atos, não do que foi pago. A certeza diz quão seguro é juntar esses atos: " +
		"contrato sem a sigla do órgão (certeza fraca) pode juntar contratos de órgãos diferentes com o mesmo número."},
		recorded(s, "entidade", s.entity))
	sdk.AddTool(srv, &sdk.Tool{Name: "fontes", Annotations: readOnly, Description: "Fontes de dados do Diário SG, " +
		"com o período coberto, a última coleta e as lacunas conhecidas. Consulte antes de concluir que algo não existe."},
		recorded(s, "fontes", s.sources))
}

func recorded[In, Out any](s *server, tool string, h sdk.ToolHandlerFor[In, Out]) sdk.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *sdk.CallToolRequest, in In) (*sdk.CallToolResult, Out, error) {
		start := s.now()
		keyID, prefix := keyFrom(req)
		if err := s.keys.RecordUse(ctx, keyID, tool); err != nil {
			s.log.Error("registro de uso do MCP", "error", err, "key_prefix", prefix, "tool", tool)
		}
		res, out, err := h(ctx, req, in)
		s.log.Info("mcp", "tool", tool, "key_prefix", prefix, "error", err != nil, "duration_ms", s.now().Sub(start).Milliseconds())
		return res, out, s.publicError(err, tool)
	}
}

func (s *server) publicError(err error, tool string) error {
	if err == nil {
		return nil
	}
	for _, known := range []error{domain.ErrNotFound, domain.ErrInvalidFilter, domain.ErrInvalidCNPJ, domain.ErrInvalidInput, errNegativeValue} {
		if errors.Is(err, known) {
			return err
		}
	}
	s.log.Error("ferramenta do MCP", "tool", tool, "error", err)
	return errInternal
}

func (s *server) search(ctx context.Context, _ *sdk.CallToolRequest, in searchInput) (*sdk.CallToolResult, searchOutput, error) {
	f, err := filterOf(in)
	if err != nil {
		return nil, searchOutput{}, err
	}
	res, err := s.searchActs.Execute(ctx, f)
	if err != nil {
		return nil, searchOutput{}, err
	}
	cov, err := s.coverage(ctx)
	if err != nil {
		return nil, searchOutput{}, err
	}
	out := searchOutput{Total: res.Total, Limit: res.Limit, Offset: res.Offset, Acts: make([]actSummaryDTO, 0, len(res.Hits)), Coverage: cov}
	for _, h := range res.Hits {
		out.Acts = append(out.Acts, summaryOf(h, s.webURL))
	}
	return nil, out, nil
}

func filterOf(in searchInput) (domain.ActFilter, error) {
	f := domain.ActFilter{Query: in.Query, Type: domain.ActType(in.Type), Organ: in.Organ, Limit: in.Limit, Offset: in.Offset}
	if f.Limit <= 0 {
		f.Limit = defaultSearchLimit
	}
	f.Limit = min(f.Limit, maxActsPerCall)
	var err error
	if f.From, err = parseDate(in.From); err != nil {
		return f, err
	}
	if f.To, err = parseDate(in.To); err != nil {
		return f, err
	}
	if f.MinCents, err = reaisToCents(in.MinValue); err != nil {
		return f, err
	}
	f.MaxCents, err = reaisToCents(in.MaxValue)
	return f, err
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return t, domain.ErrInvalidFilter
	}
	return t, nil
}

func (s *server) read(ctx context.Context, _ *sdk.CallToolRequest, in readInput) (*sdk.CallToolResult, readOutput, error) {
	g, a, err := s.readAct.Execute(ctx, in.GazetteID, in.Position)
	if err != nil {
		return nil, readOutput{}, err
	}
	cov, err := s.coverage(ctx)
	if err != nil {
		return nil, readOutput{}, err
	}
	c := citableAct(g, a)
	src := sourceOf(c, s.webURL)
	src.CollectedAt = timestampOrEmpty(g.IndexedAt)
	return nil, readOutput{
		GazetteID: g.ID, Position: a.Position, EditionNumber: g.EditionNumber, PublishedAt: g.PublishedAt.Format(time.DateOnly),
		IsExtra: g.IsExtra, Type: string(a.Type), Organ: a.Organ, OrganName: domain.OrganName(a.Organ), Title: a.Title, Text: a.Body,
		Pages:    pageRange(a.PageStart, a.PageEnd),
		Warnings: domain.ActWarnings(a.Title, domain.IsTitleOnly(a.Title, a.Body), a.PageStart, a.PageEnd),
		Citation: formatCitation(c, s.webURL, s.now()), Sources: []sourceDTO{src}, Coverage: cov,
	}, nil
}

func (s *server) entity(ctx context.Context, _ *sdk.CallToolRequest, in entityInput) (*sdk.CallToolResult, entityOutput, error) {
	kind := domain.EntityKind(in.Kind)
	if in.Kind == "" {
		kind = domain.EntityCNPJ
	}
	if !domain.IsLinkedKind(kind) {
		return nil, entityOutput{}, domain.ErrInvalidInput
	}
	report, err := s.getEntity.Execute(ctx, kind, in.Number)
	if err != nil {
		return nil, entityOutput{}, err
	}
	cov, err := s.coverage(ctx)
	if err != nil {
		return nil, entityOutput{}, err
	}
	out := entityOutput{Kind: string(report.Kind), Key: report.Key, Certainty: string(report.Certainty),
		Warning: certaintyWarning(report.Certainty), TotalActs: report.TotalActs, TotalCents: report.TotalCents,
		CountByType: map[string]int{}, Acts: make([]actSummaryDTO, 0, min(len(report.Acts), maxActsPerCall)), Coverage: cov}
	for t, n := range report.CountByType {
		out.CountByType[string(t)] = n
	}
	for _, h := range report.Acts[:min(len(report.Acts), maxActsPerCall)] {
		out.Acts = append(out.Acts, summaryOf(h, s.webURL))
	}
	return nil, out, nil
}

func certaintyWarning(c domain.Certainty) string {
	if c != domain.CertaintyWeak {
		return ""
	}
	return "Número de contrato sem a sigla do órgão: estes atos podem ser de contratos diferentes, de órgãos diferentes, com o mesmo número e ano. Confira o órgão em cada ato."
}

func (s *server) sources(ctx context.Context, _ *sdk.CallToolRequest, _ sourcesInput) (*sdk.CallToolResult, sourcesOutput, error) {
	c, err := s.cachedCoverage(ctx)
	if err != nil {
		return nil, sourcesOutput{}, err
	}
	cov := coverageOf(c)
	return nil, sourcesOutput{Sources: []sourceCoverageDTO{{
		Name: sourceName, URL: sourceSite, From: cov.From, To: cov.To, Gazettes: c.Gazettes, Acts: c.Acts,
		LastCollected: cov.LastCollected, LastRun: lastRunOf(c.LastRun), Gaps: cov.Gaps,
	}}}, nil
}

func lastRunOf(r domain.FetchRun) *lastRunDTO {
	if r.ID == "" {
		return nil
	}
	return &lastRunDTO{FinishedAt: timestampOrEmpty(r.FinishedAt), From: r.RequestedFrom.Format(time.DateOnly),
		To: r.RequestedTo.Format(time.DateOnly), Found: r.Found, Stored: r.Stored, Failed: r.Failed, Error: r.Error}
}

func (s *server) coverage(ctx context.Context) (coverageDTO, error) {
	c, err := s.cachedCoverage(ctx)
	if err != nil {
		return coverageDTO{}, err
	}
	return coverageOf(c), nil
}
