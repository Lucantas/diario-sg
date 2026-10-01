package ocrcache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

const prefix = "ocr/"

type Objects interface {
	Get(ctx context.Context, name string) (io.ReadCloser, error)
	Put(ctx context.Context, name, contentType string, body io.Reader) error
}

type Store struct {
	objects  Objects
	readOnly bool
}

type entry struct {
	Pages map[int]string `json:"pages"`
}

func New(objects Objects) Store { return Store{objects: objects} }

func (s Store) ReadOnly() Store {
	s.readOnly = true
	return s
}

func (s Store) Load(ctx context.Context, key string) (map[int]string, error) {
	rc, err := s.objects.Get(ctx, objectName(key))
	if errors.Is(err, gcp.ErrObjectNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var e entry
	if err := json.NewDecoder(rc).Decode(&e); err != nil {
		return nil, err
	}
	return e.Pages, nil
}

func (s Store) Save(ctx context.Context, key string, pages map[int]string) error {
	if s.readOnly {
		return nil
	}
	body, err := json.Marshal(entry{Pages: pages})
	if err != nil {
		return err
	}
	return s.objects.Put(ctx, objectName(key), "application/json", bytes.NewReader(body))
}

func objectName(key string) string { return prefix + key + ".json" }
