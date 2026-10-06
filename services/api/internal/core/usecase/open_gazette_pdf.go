package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

func openGazettePDF(ctx context.Context, s ports.FileStorage, src ports.SourcePDF, g domain.Gazette) (io.ReadCloser, error) {
	body, err := s.Get(ctx, g.StoragePath)
	if err == nil {
		return body, nil
	}
	if g.SourceURL == "" || src == nil {
		return nil, fmt.Errorf("pdf da edição %s: %w", g.ID, err)
	}
	body, srcErr := src.Open(ctx, g.SourceURL)
	if srcErr != nil {
		return nil, fmt.Errorf("pdf da edição %s: storage: %v; fonte: %w", g.ID, err, srcErr)
	}
	return body, nil
}
