package nostore

import (
	"context"
	"fmt"
	"io"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

type Store struct{}

func (Store) Get(_ context.Context, name string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("storage get %q: %w", name, gcp.ErrObjectNotFound)
}

func (Store) Put(context.Context, string, string, io.Reader) error { return nil }
