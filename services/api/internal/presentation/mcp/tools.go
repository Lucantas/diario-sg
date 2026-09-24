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
	maxValuesPerAct    = 20
)

var errInternal = errors.New("erro interno; tente de novo em instantes")

type searchInput struct {
	Query    string  `json:"consulta,omitempty" jsonschema:"termos da busca em português; aceita \"frase exata\", OU e -excluir"`
	Diario   string  `json:"diario,omitempty" jsonschema:"diario_prefeitura ou diario_camara; vazio busca nos dois"`
	Type     string  `json:"tipo,omitempty" jsonschema:"tipo do ato: nomeacao, exoneracao, contrato, aditivo, licitacao, dispensa, decreto, lei, portaria, resolucao, despacho, edital, ata, corrigenda, prestacao_contas ou outro"`
	Organ    string  `json:"orgao,omitempty" jsonschema:"sigla do órgão da Prefeitura, como SEMED"`
	From     string  `json:"de,omitempty" jsonschema:"data inicial da edição, AAAA-MM-DD"`
	To       string  `json:"ate,omitempty" jsonschema:"data final da edição, AAAA-MM-DD"`
	MinValue float64 `json:"valor_min,omitempty" jsonschema:"só atos que citam ao menos um valor a partir deste, em reais"`
	MaxValue float64 `json:"valor_max,omitempty" jsonschema:"só atos que citam ao menos um valor até este, em reais"`
	Modality string  `json:"modalidade,omitempty" jsonschema:"modalidade da contratação citada no ato: dispensa, inexigibilidade, pregao, concorrencia, tomada_de_precos, convite, chamamento_publico, credenciamento, adesao_ata ou leilao"`
	MainMin  float64 `json:"valor_principal_min,omitempty" jsonschema:"só atos cujo valor principal (global, total, do contrato) é a partir deste, em reais"`
	MainMax  float64 `json:"valor_principal_max,omitempty" jsonschema:"só atos cujo valor principal é até este, em reais"`
	Limit    int     `json:"limite,omitempty" jsonschema:"atos por página, de 1 a 20 (padrão 10)"`
	Offset   int     `json:"deslocamento,omitempty" jsonschema:"quantos atos pular, para paginar"`
}

type searchOutput struct {
	Total    int             `json:"total"`
	Limit    int             `json:"limite"`
	Offset   int             `json:"deslocamento"`
	Acts     []actSummaryDTO `json:"atos"`
	Coverage []coverageDTO   `json:"cobertura,omitempty"`
	Alerts   []string        `json:"alertas_coleta,omitempty"`
}

type readInput struct {
	GazetteID string `json:"edicao_id" jsonschema:"edicao_id do ato, como veio de buscar_atos"`
	Position  int    `json:"posicao" jsonschema:"posicao do ato na edição, como veio de buscar_atos"`
}

type readOutput struct {
	GazetteID     string      `json:"edicao_id"`
	Position      int         `json:"posicao"`
	Diario        string      `json:"diario"`
	EditionNumber string      `json:"edicao"`
	PublishedAt   string      `json:"data"`
	IsExtra       bool        `json:"extra"`
	Type          string      `json:"tipo"`
	Organ         string      `json:"orgao"`
	OrganName     string      `json:"orgao_nome"`
	Title         string      `json:"titulo"`
	Modality      string      `json:"modalidade,omitempty"`
	MainValue     int64       `json:"valor_principal_centavos,omitempty"`
	Text          string      `json:"texto"`
	Parties       []partyDTO  `json:"partes,omitempty"`
	Quota         *quotaDTO   `json:"cota_parlamentar,omitempty"`
	Pages         string      `json:"paginas"`
	Warnings      []string    `json:"avisos"`
	Citation      string      `json:"citacao"`
	Sources       []sourceDTO `json:"fontes"`
	Alerts        []string    `json:"alertas_coleta,omitempty"`
}

type entityInput struct {
	Kind   string `json:"tipo,omitempty" jsonschema:"cnpj (padrão), processo ou contrato"`
	Number string `json:"numero" jsonschema:"o CNPJ, o número do processo (como 06.10981/2025-7 ou 8421/2023) ou do contrato (como 55/2026 ou 30/FMS/2011)"`
	Diario string `json:"diario,omitempty" jsonschema:"diario_prefeitura ou diario_camara; vazio procura nos dois"`
}

type entityOutput struct {
	Kind        string          `json:"tipo"`
	Key         string          `json:"chave"`
	PublicBody  string          `json:"orgao_publico,omitempty"`
	Certainty   string          `json:"certeza,omitempty"`
	Warning     string          `json:"aviso,omitempty"`
	TotalActs   int             `json:"total_atos"`
	TotalCents  int64           `json:"soma_valores_centavos"`
	CountByType map[string]int  `json:"atos_por_tipo"`
	ByProcess   []processDTO    `json:"por_processo,omitempty"`
	ProcessSum  int64           `json:"soma_maior_valor_por_processo_centavos,omitempty"`
	NoProcess   int             `json:"atos_sem_processo,omitempty"`
	Acts        []actSummaryDTO `json:"atos_recentes"`
	Coverage    []coverageDTO   `json:"cobertura"`
	Alerts      []string        `json:"alertas_coleta,omitempty"`
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
	Diario        string      `json:"diario"`
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
	sdk.AddTool(srv, &sdk.Tool{Name: "buscar_atos", Annotations: readOnly, Description: "Busca atos nos Diários Oficiais de São Gonçalo, " +
		"da Prefeitura e da Câmara (nomeações, contratos, licitações, dispensas, decretos, resoluções…). " +
		"Filtre por diario para ver só um dos dois. Os termos encontrados vêm entre ⟦ ⟧ no trecho. " +
		"Use ler_ato com edicao_id e posicao para o texto completo. Resultado paginado, até 20 atos por chamada. " +
		"A cobertura e as lacunas vêm só na primeira página (deslocamento 0). Cada ato traz até 20 valores citados " +
		"(valores_total diz quantos são) e, com valor_min ou valor_max, os que caíram na faixa. " +
		"alertas_coleta aparece quando a última coleta de um diário falhou. " +
		"modalidade (dispensa, inexigibilidade, pregao…) e valor_principal_centavos (valor global, total ou do contrato) " +
		"são lidos do texto e podem faltar; o tipo do ato não muda, então um extrato de contrato por dispensa tem tipo contrato e modalidade dispensa."},
		recorded(s, "buscar_atos", s.search))
	sdk.AddTool(srv, &sdk.Tool{Name: "ler_ato", Annotations: readOnly, Description: "Texto completo de um ato, " +
		"com citação pronta (ABNT), link da edição oficial na página do ato e cópia arquivada com SHA-256. " +
		"partes lista cada CNPJ com o nome provável lido do texto ao lado do número (confira no texto); " +
		"cota_parlamentar traz vereador, mês de referência e valor dos termos de CEAPM da Câmara."},
		recorded(s, "ler_ato", s.read))
	sdk.AddTool(srv, &sdk.Tool{Name: "entidade", Annotations: readOnly, Description: "Atos dos Diários (Prefeitura e Câmara) que citam um CNPJ, " +
		"um processo ou um contrato: total de atos, contagem por tipo, soma dos valores citados nesses atos e os 20 mais recentes. " +
		"A soma é do que aparece no texto dos atos, não do que foi pago. A certeza diz quão seguro é juntar esses atos: " +
		"contrato sem a sigla do órgão (certeza fraca) pode juntar contratos de órgãos diferentes com o mesmo número. " +
		"por_processo agrupa os atos pelos processos citados, com o maior valor de cada um; " +
		"soma_maior_valor_por_processo_centavos soma esses maiores valores, para não contar a mesma contratação várias vezes " +
		"(extrato, aditivo e homologação citam o mesmo valor). " +
		"orgao_publico diz quando o CNPJ é do Município, de uma fundação ou fundo municipal, do SG-PREVI ou da Câmara: não é fornecedor."},
		recorded(s, "entidade", s.entity))
	sdk.AddTool(srv, &sdk.Tool{Name: "agrupar", Annotations: readOnly, Description: groupDescription},
		recorded(s, "agrupar", s.group))
	sdk.AddTool(srv, &sdk.Tool{Name: "pagina_original", Annotations: readOnly, Description: pageDescription},
		recorded(s, "pagina_original", s.page))
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
	cs, err := s.cachedCoverage(ctx)
	if err != nil {
		return nil, searchOutput{}, err
	}
	out := searchOutput{Total: res.Total, Limit: res.Limit, Offset: res.Offset, Acts: make([]actSummaryDTO, 0, len(res.Hits)),
		Alerts: collectionAlerts(cs)}
	if f.Offset == 0 {
		out.Coverage = coveragesOf(cs)
	}
	for _, h := range res.Hits {
		act := summaryOf(h, s.webURL)
		act.InRange = valuesInRange(h.ValuesCents, f.MinCents, f.MaxCents)
		out.Acts = append(out.Acts, act)
	}
	return nil, out, nil
}

func filterOf(in searchInput) (domain.ActFilter, error) {
	f := domain.ActFilter{Query: in.Query, Source: in.Diario, Type: domain.ActType(in.Type), Organ: in.Organ,
		Modality: domain.Modality(in.Modality), Limit: in.Limit, Offset: in.Offset}
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
	if f.MainMinCents, err = reaisToCents(in.MainMin); err != nil {
		return f, err
	}
	if f.MainMaxCents, err = reaisToCents(in.MainMax); err != nil {
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
	cs, err := s.cachedCoverage(ctx)
	if err != nil {
		return nil, readOutput{}, err
	}
	c := citableAct(g, a)
	src := sourceOf(c, s.webURL)
	src.CollectedAt = timestampOrEmpty(g.IndexedAt)
	return nil, readOutput{
		GazetteID: g.ID, Position: a.Position, Diario: domain.SourceOrDefault(g.Source), EditionNumber: g.EditionNumber, PublishedAt: g.PublishedAt.Format(time.DateOnly),
		IsExtra: g.IsExtra, Type: string(a.Type), Organ: a.Organ, OrganName: domain.OrganName(a.Organ), Title: a.Title, Text: a.Body,
		Modality: string(a.Modality), MainValue: a.MainValueCents, Parties: partiesOf(a.Body), Quota: quotaOf(a),
		Pages:    pageRange(a.PageStart, a.PageEnd),
		Warnings: domain.ActWarnings(domain.WarningFactsOf(a)),
		Citation: formatCitation(c, s.webURL, s.now()), Sources: []sourceDTO{src}, Alerts: collectionAlerts(cs),
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
	report, err := s.getEntity.Execute(ctx, kind, in.Number, in.Diario)
	if err != nil {
		return nil, entityOutput{}, err
	}
	cs, err := s.cachedCoverage(ctx)
	if err != nil {
		return nil, entityOutput{}, err
	}
	out := entityOutput{Kind: string(report.Kind), Key: report.Key, PublicBody: publicBodyOf(report), Certainty: string(report.Certainty),
		Warning: certaintyWarning(report), TotalActs: report.TotalActs, TotalCents: report.TotalCents,
		CountByType: map[string]int{}, Acts: make([]actSummaryDTO, 0, min(len(report.Acts), maxActsPerCall)),
		Coverage: coveragesOf(cs), Alerts: collectionAlerts(cs),
		ByProcess: processesOf(report.ByProcess), ProcessSum: report.SumOfProcessMaxCents(), NoProcess: report.ActsWithoutProcess}
	for t, n := range report.CountByType {
		out.CountByType[string(t)] = n
	}
	for _, h := range report.Acts[:min(len(report.Acts), maxActsPerCall)] {
		out.Acts = append(out.Acts, summaryOf(h, s.webURL))
	}
	return nil, out, nil
}

type processDTO struct {
	Key           string `json:"processo"`
	Acts          int    `json:"atos"`
	MaxValueCents int64  `json:"maior_valor_centavos"`
	First         string `json:"primeira_data"`
	Last          string `json:"ultima_data"`
}

func processesOf(ps []domain.ProcessSummary) []processDTO {
	out := make([]processDTO, 0, len(ps))
	for _, p := range ps {
		out = append(out, processDTO{Key: p.Key, Acts: p.Acts, MaxValueCents: p.MaxValueCents,
			First: p.First.Format(time.DateOnly), Last: p.Last.Format(time.DateOnly)})
	}
	return out
}

func publicBodyOf(r domain.EntityReport) string {
	if r.Kind != domain.EntityCNPJ {
		return ""
	}
	name, _ := domain.PublicBody(r.Key)
	return name
}

func certaintyWarning(r domain.EntityReport) string {
	switch {
	case r.Kind != domain.EntityCNPJ && r.Sources > 1:
		return "Este número aparece no Diário da Prefeitura e no da Câmara, que numeram processos e contratos cada um à sua maneira: " +
			"podem ser registros diferentes. Use diario para ver só um dos dois e confira o campo diario de cada ato."
	case r.Certainty == domain.CertaintyWeak:
		return "Número de contrato sem a sigla do órgão: estes atos podem ser de contratos diferentes, de órgãos diferentes, com o mesmo número e ano. Confira o órgão em cada ato."
	}
	return ""
}

func (s *server) sources(ctx context.Context, _ *sdk.CallToolRequest, _ sourcesInput) (*sdk.CallToolResult, sourcesOutput, error) {
	cs, err := s.cachedCoverage(ctx)
	if err != nil {
		return nil, sourcesOutput{}, err
	}
	out := sourcesOutput{Sources: make([]sourceCoverageDTO, 0, len(cs))}
	for _, c := range cs {
		cov := coverageOf(c)
		out.Sources = append(out.Sources, sourceCoverageDTO{
			Diario: c.Source, Name: cov.Name, URL: sourceSites[c.Source], From: cov.From, To: cov.To, Gazettes: c.Gazettes, Acts: c.Acts,
			LastCollected: cov.LastCollected, LastRun: lastRunOf(c.LastRun), Gaps: cov.Gaps,
		})
	}
	return nil, out, nil
}

func lastRunOf(r domain.FetchRun) *lastRunDTO {
	if r.ID == "" {
		return nil
	}
	return &lastRunDTO{FinishedAt: timestampOrEmpty(r.FinishedAt), From: r.RequestedFrom.Format(time.DateOnly),
		To: r.RequestedTo.Format(time.DateOnly), Found: r.Found, Stored: r.Stored, Failed: r.Failed, Error: r.Error}
}
