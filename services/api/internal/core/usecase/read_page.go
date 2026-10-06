package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type PageText struct {
	Gazette   domain.Gazette
	Page      int
	Pages     int
	Text      string
	ReadByOCR bool
}

type ReadPage struct {
	gazettes ports.GazetteRepository
	storage  ports.FileStorage
	source   ports.SourcePDF
	pages    ports.PageExtractor
}

func NewReadPage(g ports.GazetteRepository, s ports.FileStorage, src ports.SourcePDF, p ports.PageExtractor) *ReadPage {
	return &ReadPage{gazettes: g, storage: s, source: src, pages: p}
}

func (uc *ReadPage) Execute(ctx context.Context, gazetteID string, page int) (PageText, error) {
	if page < 1 {
		return PageText{}, domain.ErrInvalidInput
	}
	g, err := uc.gazettes.FindByID(ctx, gazetteID)
	if err != nil {
		return PageText{}, err
	}
	body, err := openGazettePDF(ctx, uc.storage, uc.source, g)
	if err != nil {
		return PageText{}, err
	}
	defer body.Close()
	extracted, pages, err := uc.pages.ExtractPage(ctx, body, domain.SourceOrDefault(g.Source), page)
	if err != nil {
		return PageText{}, err
	}
	return PageText{Gazette: g, Page: page, Pages: pages, Text: extracted.Text, ReadByOCR: len(extracted.OCRPages) > 0}, nil
}
