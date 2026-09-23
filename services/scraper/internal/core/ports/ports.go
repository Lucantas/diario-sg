package ports

import (
	"context"
	"io"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

type EditionSource interface {
	ListEditions(ctx context.Context, from, to time.Time) ([]domain.Edition, error)
	Download(ctx context.Context, e domain.Edition) (io.ReadCloser, error)
}

type ObjectStorage interface {
	Exists(ctx context.Context, path string) (bool, error)
	Put(ctx context.Context, path, contentType string, r io.Reader) error
}

type EventPublisher interface {
	EditionFetched(ctx context.Context, e domain.FetchedEdition) error
	RunCompleted(ctx context.Context, r domain.FetchRun) error
}
