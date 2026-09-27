package mcp

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const (
	patternsTTL             = 10 * time.Minute
	patternsRefreshTimeout  = 2 * time.Minute
	findingsPerPatternBrief = 3
	maxFindingsPerCall      = 20
	maxActsPerFinding       = 10
)

const patternsDescription = "Padrões para verificar nos atos e nas fontes cruzadas (fracionamento de dispensa, aditivo acima do limite, " +
	"emergencial renovada, pico de nomeações na eleição, empresa nova contratada, capital menor que o contrato, sócio ou endereço em comum, " +
	"sancionado contratado, pago sem publicação, contrato sem pagamento, pago acima do anunciado, contrato do PNCP sem extrato). " +
	"Padrão não é irregularidade: é pista para conferir, e cada padrão traz a regra e a ressalva. " +
	"Sem argumentos: o catálogo, com o número de achados e os 3 primeiros de cada padrão, sem a lista de atos. " +
	"padrao (o id) traz até 20 achados daquele padrão, e pular avança. " +
	"cnpj, processo ou contrato (um só) traz os achados que citam aquela entidade, com a mesma chave da ferramenta entidade. " +
	"Cada achado traz entidades (CNPJ, processo, contrato), até 10 atos que o acionaram (atos_total diz quantos são), com o link da edição oficial na página e a cópia arquivada, e, quando há, a busca no site. " +
	"Use ler_ato com edicao_id e posicao para o texto de cada ato. " +
	"Não há pessoa física: sócio em comum aparece sem o nome da pessoa."

type patternsInput struct {
	Pattern  string `json:"padrao,omitempty" jsonschema:"id do padrão, como fracionamento_dispensa ou pago_sem_publicacao"`
	CNPJ     string `json:"cnpj,omitempty" jsonschema:"CNPJ, com ou sem pontuação"`
	Processo string `json:"processo,omitempty" jsonschema:"número do processo, como 06.10981/2025-7"`
	Contrato string `json:"contrato,omitempty" jsonschema:"número do contrato, como 55/2026 ou 30/FMS/2011"`
	Skip     int    `json:"pular,omitempty" jsonschema:"quantos achados pular, com padrao"`
}

type findingEntityDTO struct {
	Kind  string `json:"tipo"`
	Key   string `json:"chave"`
	Label string `json:"rotulo"`
}

type findingActDTO struct {
	GazetteID string `json:"edicao_id"`
	Position  int    `json:"posicao"`
	Diario    string `json:"diario"`
	Edition   string `json:"edicao"`
	Date      string `json:"data"`
	Pages     string `json:"paginas"`
	Type      string `json:"tipo"`
	Title     string `json:"titulo"`
	URL       string `json:"url"`
	Archived  string `json:"copia_arquivada"`
}

type findingDTO struct {
	Pattern   string             `json:"padrao"`
	Title     string             `json:"titulo"`
	Detail    string             `json:"detalhe"`
	Entities  []findingEntityDTO `json:"entidades"`
	Acts      []findingActDTO    `json:"atos"`
	ActsTotal int                `json:"atos_total"`
	LinkLabel string             `json:"link_rotulo,omitempty"`
	Link      string             `json:"link,omitempty"`
}

type patternDTO struct {
	ID       string `json:"id"`
	Title    string `json:"titulo"`
	Rule     string `json:"regra"`
	Caveat   string `json:"ressalva"`
	Findings int    `json:"achados_total"`
}

type patternsOutput struct {
	Patterns []patternDTO `json:"padroes"`
	Findings []findingDTO `json:"achados"`
	Total    int          `json:"achados_encontrados"`
	Entity   string       `json:"entidade,omitempty"`
}

type patternsSnapshot struct {
	reports []domain.PatternReport
	acts    map[string]domain.ActHit
	at      time.Time
}

func (s *server) patterns(ctx context.Context, _ *sdk.CallToolRequest, in patternsInput) (*sdk.CallToolResult, patternsOutput, error) {
	kind, key, err := patternEntity(in)
	if err != nil {
		return nil, patternsOutput{}, err
	}
	catalog := kind == "" && in.Pattern == ""
	if in.Skip < 0 || (catalog && in.Skip > 0) {
		return nil, patternsOutput{}, fmt.Errorf("%w: pular não pode ser negativo e só vale com padrao, cnpj, processo ou contrato", domain.ErrInvalidInput)
	}
	snap, err := s.cachedPatterns(ctx)
	if err != nil {
		return nil, patternsOutput{}, err
	}
	reports, err := selectReports(snap.reports, in.Pattern)
	if err != nil {
		return nil, patternsOutput{}, err
	}
	out := patternsOutput{Patterns: make([]patternDTO, 0, len(reports)), Findings: []findingDTO{}}
	type match struct {
		id domain.PatternID
		f  domain.Finding
	}
	var matched []match
	for _, rep := range reports {
		dto := patternDTO{ID: string(rep.Pattern.ID), Title: rep.Pattern.Title, Rule: rep.Pattern.Rule, Caveat: rep.Pattern.Caveat}
		var own []domain.Finding
		for _, f := range rep.Findings {
			if kind == "" || f.Mentions(kind, key) {
				own = append(own, f)
			}
		}
		dto.Findings = len(own)
		out.Total += len(own)
		if kind != "" && len(own) == 0 {
			continue
		}
		out.Patterns = append(out.Patterns, dto)
		if catalog {
			for _, f := range own[:min(len(own), findingsPerPatternBrief)] {
				brief := s.findingOf(rep.Pattern.ID, f, snap.acts)
				brief.Acts = []findingActDTO{}
				out.Findings = append(out.Findings, brief)
			}
			continue
		}
		for _, f := range own {
			matched = append(matched, match{rep.Pattern.ID, f})
		}
	}
	if kind != "" {
		out.Entity = string(kind) + " " + key
	}
	page := matched[min(in.Skip, len(matched)):]
	for _, m := range page[:min(len(page), maxFindingsPerCall)] {
		out.Findings = append(out.Findings, s.findingOf(m.id, m.f, snap.acts))
	}
	return nil, out, nil
}

func patternEntity(in patternsInput) (domain.EntityKind, string, error) {
	var kind domain.EntityKind
	var raw string
	for k, v := range map[domain.EntityKind]string{domain.EntityCNPJ: in.CNPJ, domain.EntityProcesso: in.Processo, domain.EntityContrato: in.Contrato} {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if kind != "" {
			return "", "", fmt.Errorf("%w: informe só um entre cnpj, processo e contrato", domain.ErrInvalidInput)
		}
		kind, raw = k, v
	}
	if kind == "" {
		return "", "", nil
	}
	key, err := domain.ParseEntityInput(kind, raw)
	if err != nil {
		return "", "", err
	}
	return kind, key, nil
}

func selectReports(reports []domain.PatternReport, id string) ([]domain.PatternReport, error) {
	if id == "" {
		return reports, nil
	}
	for _, r := range reports {
		if string(r.Pattern.ID) == id {
			return []domain.PatternReport{r}, nil
		}
	}
	ids := make([]string, len(reports))
	for i, r := range reports {
		ids[i] = string(r.Pattern.ID)
	}
	return nil, fmt.Errorf("%w: padrão %q não existe; use um de %s", domain.ErrInvalidInput, id, strings.Join(ids, ", "))
}

func (s *server) findingOf(id domain.PatternID, f domain.Finding, acts map[string]domain.ActHit) findingDTO {
	dto := findingDTO{Pattern: string(id), Title: f.Title, Detail: f.Detail, Entities: make([]findingEntityDTO, 0, len(f.Entities)),
		Acts: []findingActDTO{}, ActsTotal: len(f.ActIDs)}
	for _, e := range f.Entities {
		dto.Entities = append(dto.Entities, findingEntityDTO{Kind: string(e.Kind), Key: e.Key, Label: e.Label})
	}
	for _, actID := range f.ActIDs {
		h, ok := acts[actID]
		if !ok || len(dto.Acts) == maxActsPerFinding {
			continue
		}
		src := sourceOf(citableHit(h), s.webURL)
		dto.Acts = append(dto.Acts, findingActDTO{GazetteID: h.GazetteID, Position: h.Position, Diario: domain.SourceOrDefault(h.Source),
			Edition: h.EditionNumber, Date: h.PublishedAt.Format(time.DateOnly), Pages: pageRange(h.PageStart, h.PageEnd), Type: string(h.Type),
			Title: h.Title, URL: src.URL, Archived: src.ArchivedCopy})
	}
	switch {
	case f.Search != nil:
		dto.LinkLabel = "Ver os atos na busca"
		dto.Link = s.webURL + "/?" + url.Values{"tipo": {string(f.Search.Type)}, "de": {f.Search.From.Format(time.DateOnly)},
			"ate": {f.Search.To.Format(time.DateOnly)}, "fonte": {f.Search.Source}}.Encode()
	case f.Link != nil:
		dto.LinkLabel, dto.Link = f.Link.Label, f.Link.URL
		if strings.HasPrefix(dto.Link, "/") {
			dto.Link = s.webURL + dto.Link
		}
	}
	return dto
}

func (s *server) cachedPatterns(ctx context.Context) (patternsSnapshot, error) {
	s.patternsMu.Lock()
	if s.patternsCache.reports != nil && s.now().Sub(s.patternsCache.at) < patternsTTL {
		defer s.patternsMu.Unlock()
		return s.patternsCache, nil
	}
	done := s.patternsRun
	if done == nil {
		done = make(chan struct{})
		s.patternsRun = done
		go s.refreshPatterns(context.WithoutCancel(ctx), done)
	}
	s.patternsMu.Unlock()
	select {
	case <-done:
	case <-ctx.Done():
		return patternsSnapshot{}, ctx.Err()
	}
	s.patternsMu.Lock()
	defer s.patternsMu.Unlock()
	if s.patternsErr != nil {
		return patternsSnapshot{}, s.patternsErr
	}
	return s.patternsCache, nil
}

func (s *server) refreshPatterns(ctx context.Context, done chan struct{}) {
	ctx, cancel := context.WithTimeout(ctx, patternsRefreshTimeout)
	defer cancel()
	reports, acts, err := s.listPatterns.Execute(ctx)
	s.patternsMu.Lock()
	defer s.patternsMu.Unlock()
	s.patternsErr = err
	if err == nil {
		s.patternsCache = patternsSnapshot{reports: reports, acts: acts, at: s.now()}
	}
	s.patternsRun = nil
	close(done)
}
