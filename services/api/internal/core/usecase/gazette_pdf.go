package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type GetGazettePDF struct {
	gazettes ports.GazetteRepository
	storage  ports.FileStorage
}

type GazettePDF struct {
	Body        io.ReadCloser
	RedirectURL string
}

func NewGetGazettePDF(g ports.GazetteRepository, s ports.FileStorage) *GetGazettePDF {
	return &GetGazettePDF{gazettes: g, storage: s}
}

func (uc *GetGazettePDF) Gazette(ctx context.Context, id string) (domain.Gazette, error) {
	return uc.gazettes.FindByID(ctx, id)
}

func (uc *GetGazettePDF) Open(ctx context.Context, g domain.Gazette) (GazettePDF, error) {
	body, err := uc.storage.Get(ctx, g.StoragePath)
	if err == nil {
		return GazettePDF{Body: body}, nil
	}
	if g.SourceURL != "" {
		return GazettePDF{RedirectURL: g.SourceURL}, nil
	}
	return GazettePDF{}, fmt.Errorf("pdf da edição %s: %w", g.ID, err)
}
