package mcp

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const (
	sourceName = "Diário Oficial de São Gonçalo"
	sourceSite = "https://do.pmsg.rj.gov.br/"
	brDate     = "02/01/2006"
)

var knownGaps = []string{
	"Só o Diário Oficial da Prefeitura; o Diário da Câmara ainda não é coletado.",
	"Edição em PDF só com imagem (escaneada) não tem o texto lido.",
	"A separação em atos é automática e pode errar; os avisos de cada ato dizem onde desconfiar.",
	"Ausência de resultado não prova que o ato não existe: confira a edição original.",
}

type sourceDTO struct {
	Name         string `json:"nome"`
	URL          string `json:"url"`
	ArchivedCopy string `json:"copia_arquivada"`
	Page         int    `json:"pagina,omitempty"`
	SHA256       string `json:"sha256"`
	CollectedAt  string `json:"coletado_em,omitempty"`
}

type coverageDTO struct {
	Source        string   `json:"fonte"`
	From          string   `json:"de"`
	To            string   `json:"ate"`
	LastCollected string   `json:"ultima_coleta"`
	Gaps          []string `json:"lacunas"`
}

type actSummaryDTO struct {
	GazetteID     string      `json:"edicao_id"`
	Position      int         `json:"posicao"`
	EditionNumber string      `json:"edicao"`
	PublishedAt   string      `json:"data"`
	IsExtra       bool        `json:"extra"`
	Type          string      `json:"tipo"`
	Organ         string      `json:"orgao"`
	OrganName     string      `json:"orgao_nome"`
	Title         string      `json:"titulo"`
	Snippet       string      `json:"trecho"`
	Pages         string      `json:"paginas"`
	CNPJs         []string    `json:"cnpjs"`
	ValuesCents   []int64     `json:"valores_centavos"`
	Warnings      []string    `json:"avisos"`
	Sources       []sourceDTO `json:"fontes"`
}

func sourceOf(c citable, webURL string) sourceDTO {
	return sourceDTO{
		Name:         fmt.Sprintf("%s, edição %s de %s", sourceName, editionLabel(c.EditionNumber), c.PublishedAt.Format(brDate)),
		URL:          c.SourceURL + pageFragment(c.PageStart),
		ArchivedCopy: archivedPDFURL(webURL, c),
		Page:         c.PageStart,
		SHA256:       c.Checksum,
	}
}

func editionLabel(number string) string {
	if number == "" {
		return "s/n"
	}
	return number
}

func coverageOf(c domain.Coverage) coverageDTO {
	return coverageDTO{Source: sourceName, From: dateOrEmpty(c.First), To: dateOrEmpty(c.Last),
		LastCollected: timestampOrEmpty(c.LastIndexedAt), Gaps: knownGaps}
}

func dateOrEmpty(t time.Time) string {
	if t.Year() <= 1970 {
		return ""
	}
	return t.Format(time.DateOnly)
}

func timestampOrEmpty(t time.Time) string {
	if t.Year() <= 1970 {
		return ""
	}
	return t.In(saoPaulo).Format(time.RFC3339)
}

func citableHit(h domain.ActHit) citable {
	return citable{GazetteID: h.GazetteID, Title: h.Title, EditionNumber: h.EditionNumber, PublishedAt: h.PublishedAt,
		IsExtra: h.IsExtra, SourceURL: h.SourceURL, PageStart: h.PageStart, PageEnd: h.PageEnd, Checksum: h.Checksum}
}

func citableAct(g domain.Gazette, a domain.Act) citable {
	return citable{GazetteID: g.ID, Title: a.Title, EditionNumber: g.EditionNumber, PublishedAt: g.PublishedAt,
		IsExtra: g.IsExtra, SourceURL: g.SourceURL, PageStart: a.PageStart, PageEnd: a.PageEnd, Checksum: g.Checksum}
}

func summaryOf(h domain.ActHit, webURL string) actSummaryDTO {
	return actSummaryDTO{
		GazetteID: h.GazetteID, Position: h.Position, EditionNumber: h.EditionNumber,
		PublishedAt: h.PublishedAt.Format(time.DateOnly), IsExtra: h.IsExtra, Type: string(h.Type),
		Organ: h.Organ, OrganName: domain.OrganName(h.Organ), Title: h.Title, Snippet: h.Snippet,
		Pages: pageRange(h.PageStart, h.PageEnd), CNPJs: nonNil(h.CNPJs), ValuesCents: nonNil(h.ValuesCents),
		Warnings: domain.ActWarnings(h.Title, h.TitleOnly, h.PageStart, h.PageEnd),
		Sources:  []sourceDTO{sourceOf(citableHit(h), webURL)},
	}
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

var errNegativeValue = errors.New("valor em reais não pode ser negativo")

func reaisToCents(reais float64) (int64, error) {
	if reais < 0 || math.IsNaN(reais) || math.IsInf(reais, 0) {
		return 0, errNegativeValue
	}
	return int64(math.Round(reais * 100)), nil
}
