package mcp

import (
	"context"
	"time"
	"unicode/utf8"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const maxPageRunes = 30000

type pageInput struct {
	GazetteID string `json:"edicao_id" jsonschema:"edicao_id do ato, como veio de buscar_atos"`
	Page      int    `json:"pagina" jsonschema:"número da página no PDF, a partir de 1"`
}

type pageOutput struct {
	GazetteID     string    `json:"edicao_id"`
	Diario        string    `json:"diario"`
	EditionNumber string    `json:"edicao"`
	PublishedAt   string    `json:"data"`
	Page          int       `json:"pagina"`
	Pages         int       `json:"paginas_total"`
	Text          string    `json:"texto"`
	Truncated     bool      `json:"truncado,omitempty"`
	Source        sourceDTO `json:"fonte"`
}

const pageDescription = "Texto cru de uma página do PDF arquivado de uma edição, como o pdftotext o extrai, " +
	"com o SHA-256 do PDF e o link da página na edição oficial. Serve para conferir o que o parser leu " +
	"(separação de atos, valores, CNPJs) sem sair do MCP. As páginas vêm em paginas de cada ato."

func (s *server) page(ctx context.Context, _ *sdk.CallToolRequest, in pageInput) (*sdk.CallToolResult, pageOutput, error) {
	p, err := s.readPage.Execute(ctx, in.GazetteID, in.Page)
	if err != nil {
		return nil, pageOutput{}, err
	}
	g := p.Gazette
	c := citable{GazetteID: g.ID, EditionNumber: g.EditionNumber, PublishedAt: g.PublishedAt, IsExtra: g.IsExtra,
		SourceURL: g.SourceURL, PageStart: p.Page, PageEnd: p.Page, Checksum: g.Checksum, Source: domain.SourceOrDefault(g.Source)}
	text, truncated := truncateRunes(p.Text, maxPageRunes)
	return nil, pageOutput{GazetteID: g.ID, Diario: c.Source, EditionNumber: g.EditionNumber, PublishedAt: g.PublishedAt.Format(time.DateOnly),
		Page: p.Page, Pages: p.Pages, Text: text, Truncated: truncated, Source: sourceOf(c, s.webURL)}, nil
}

func truncateRunes(s string, n int) (string, bool) {
	if utf8.RuneCountInString(s) <= n {
		return s, false
	}
	return string([]rune(s)[:n]), true
}
