package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type failingStorage struct{}

func (failingStorage) Get(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("sem objeto")
}

type recordingSource struct {
	urls []string
	err  error
}

func (s *recordingSource) Open(_ context.Context, url string) (io.ReadCloser, error) {
	s.urls = append(s.urls, url)
	if s.err != nil {
		return nil, s.err
	}
	return io.NopCloser(strings.NewReader("%PDF-oficial")), nil
}

func TestOpenGazettePDFFallsBackToSourceURL(t *testing.T) {
	src := &recordingSource{}
	g := domain.Gazette{ID: "g1", StoragePath: "1.pdf", SourceURL: "https://oficial/1.pdf"}

	body, err := openGazettePDF(context.Background(), failingStorage{}, src, g)

	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(body)
	if string(b) != "%PDF-oficial" || len(src.urls) != 1 || src.urls[0] != g.SourceURL {
		t.Errorf("corpo %q, urls %v", b, src.urls)
	}
}

func TestOpenGazettePDFPrefersStorage(t *testing.T) {
	src := &recordingSource{}

	_, err := openGazettePDF(context.Background(), memStorage{}, src, domain.Gazette{ID: "g1", StoragePath: "1.pdf", SourceURL: "https://oficial/1.pdf"})

	if err != nil || len(src.urls) != 0 {
		t.Fatalf("não devia ir à fonte: %v %v", err, src.urls)
	}
}

func TestOpenGazettePDFWithoutSourceURLFails(t *testing.T) {
	_, err := openGazettePDF(context.Background(), failingStorage{}, &recordingSource{}, domain.Gazette{ID: "g1"})

	if err == nil {
		t.Fatal("sem storage e sem source_url deveria falhar")
	}
}
