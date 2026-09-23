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

func NewGetGazettePDF(g ports.GazetteRepository, s ports.FileStorage) *GetGazettePDF {
	return &GetGazettePDF{gazettes: g, storage: s}
}

func (uc *GetGazettePDF) Gazette(ctx context.Context, id string) (domain.Gazette, error) {
	return uc.gazettes.FindByID(ctx, id)
}

func (uc *GetGazettePDF) Open(ctx context.Context, g domain.Gazette) (io.ReadCloser, error) {
	body, err := uc.storage.Get(ctx, g.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("pdf da edição %s: %w", g.ID, err)
	}
	return body, nil
}
