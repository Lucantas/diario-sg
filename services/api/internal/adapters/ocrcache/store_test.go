package ocrcache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"testing"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

type memObjects struct {
	data    map[string][]byte
	types   map[string]string
	failGet bool
}

func (m *memObjects) Get(_ context.Context, name string) (io.ReadCloser, error) {
	if m.failGet {
		return nil, errors.New("status 500")
	}
	b, ok := m.data[name]
	if !ok {
		return nil, fmt.Errorf("get %q: %w", name, gcp.ErrObjectNotFound)
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (m *memObjects) Put(_ context.Context, name, contentType string, body io.Reader) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.data[name], m.types[name] = b, contentType
	return nil
}

func TestSavedPagesComeBackUnderTheOCRPrefix(t *testing.T) {
	objects := &memObjects{data: map[string][]byte{}, types: map[string]string{}}
	store := New(objects)
	pages := map[int]string{1: "DECRETO", 12: "PORTARIA Nº 3"}

	if err := store.Save(context.Background(), "v1/abc", pages); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(context.Background(), "v1/abc")

	if err != nil || !maps.Equal(got, pages) {
		t.Fatalf("lido: %v %v", got, err)
	}
	if objects.types["ocr/v1/abc.json"] != "application/json" {
		t.Errorf("objeto gravado: %v", objects.types)
	}
}

func TestMissingObjectIsAnEmptyCacheAndOtherFailuresAreErrors(t *testing.T) {
	objects := &memObjects{data: map[string][]byte{}}

	missing, err := New(objects).Load(context.Background(), "v1/nada")
	if err != nil || missing != nil {
		t.Errorf("ausente: %v %v", missing, err)
	}
	objects.failGet = true
	if _, err := New(objects).Load(context.Background(), "v1/nada"); err == nil {
		t.Error("falha do bucket deveria voltar como erro")
	}
}

func TestReadOnlyNeverWrites(t *testing.T) {
	objects := &memObjects{data: map[string][]byte{}, types: map[string]string{}}

	if err := New(objects).ReadOnly().Save(context.Background(), "v1/abc", map[int]string{1: "X"}); err != nil {
		t.Fatal(err)
	}

	if len(objects.data) != 0 {
		t.Errorf("gravou: %v", objects.data)
	}
}
