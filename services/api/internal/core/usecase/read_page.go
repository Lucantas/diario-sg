package usecase

import (
	"context"
	"fmt"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type PageText struct {
	Gazette domain.Gazette
	Page    int
	Pages   int
	Text    string
}

type ReadPage struct {
	gazettes ports.GazetteRepository
	storage  ports.FileStorage
	pages    ports.PageExtractor
}

func NewReadPage(g ports.GazetteRepository, s ports.FileStorage, p ports.PageExtractor) *ReadPage {
	return &ReadPage{gazettes: g, storage: s, pages: p}
}

func (uc *ReadPage) Execute(ctx context.Context, gazetteID string, page int) (PageText, error) {
	if page < 1 {
		return PageText{}, domain.ErrInvalidInput
	}
	g, err := uc.gazettes.FindByID(ctx, gazetteID)
	if err != nil {
		return PageText{}, err
	}
	body, err := uc.storage.Get(ctx, g.StoragePath)
	if err != nil {
		return PageText{}, fmt.Errorf("pdf da edição %s: %w", g.ID, err)
	}
	defer body.Close()
	text, pages, err := uc.pages.ExtractPage(ctx, body, domain.SourceOrDefault(g.Source), page)
	if err != nil {
		return PageText{}, err
	}
	return PageText{Gazette: g, Page: page, Pages: pages, Text: text}, nil
}
