// Package ports define as interfaces que o core precisa do mundo externo.
// As implementações ficam em internal/adapters (inversão de dependência).
package ports

import (
	"context"
	"io"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

// EditionSource é de onde as edições vêm (site da prefeitura, API, etc.).
type EditionSource interface {
	ListEditions(ctx context.Context, from, to time.Time) ([]domain.Edition, error)
	Download(ctx context.Context, e domain.Edition) (io.ReadCloser, error)
}

// ObjectStorage guarda os arquivos brutos.
type ObjectStorage interface {
	Exists(ctx context.Context, path string) (bool, error)
	Put(ctx context.Context, path, contentType string, r io.Reader) error
}

// EventPublisher avisa o restante do sistema.
type EventPublisher interface {
	EditionFetched(ctx context.Context, e domain.FetchedEdition) error
}
