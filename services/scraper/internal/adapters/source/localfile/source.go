package localfile

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

type Source struct {
	edition domain.Edition
	path    string
}

func New(path string, published time.Time, number, sourceURL string) (*Source, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("arquivo da edição: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("arquivo da edição: %s é um diretório", path)
	}
	if published.IsZero() {
		return nil, fmt.Errorf("data de publicação é obrigatória")
	}
	return &Source{path: path, edition: domain.Edition{Number: number, PublishedAt: published, URL: sourceURL}}, nil
}

func (s *Source) ListEditions(context.Context, time.Time, time.Time) ([]domain.Edition, error) {
	return []domain.Edition{s.edition}, nil
}

func (s *Source) Download(_ context.Context, e domain.Edition) (io.ReadCloser, error) {
	if e.URL != s.edition.URL {
		return nil, fmt.Errorf("edição desconhecida: %s", e.URL)
	}
	return os.Open(s.path)
}
