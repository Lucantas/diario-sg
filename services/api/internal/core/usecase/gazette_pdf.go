package usecase

import (
	"context"
	"io"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type GetGazettePDF struct {
	gazettes ports.GazetteRepository
	storage  ports.FileStorage
	source   ports.SourcePDF
}

func NewGetGazettePDF(g ports.GazetteRepository, s ports.FileStorage, src ports.SourcePDF) *GetGazettePDF {
	return &GetGazettePDF{gazettes: g, storage: s, source: src}
}

func (uc *GetGazettePDF) Gazette(ctx context.Context, id string) (domain.Gazette, error) {
	return uc.gazettes.FindByID(ctx, id)
}

func (uc *GetGazettePDF) Open(ctx context.Context, g domain.Gazette) (io.ReadCloser, error) {
	return openGazettePDF(ctx, uc.storage, uc.source, g)
}
