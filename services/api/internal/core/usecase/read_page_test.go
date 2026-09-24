package usecase

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type onePage struct {
	gotSource string
	gotPage   int
}

func (o *onePage) ExtractPage(_ context.Context, r io.Reader, source string, page int) (string, int, error) {
	b, _ := io.ReadAll(r)
	o.gotSource, o.gotPage = source, page
	return string(b) + " página", 4, nil
}

func TestReadPageExtractsFromTheArchivedPDF(t *testing.T) {
	gaz := newMemGazettes()
	g := &domain.Gazette{Checksum: "abc", StoragePath: "x.pdf", Source: domain.SourceDiarioCamara, PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)}
	_ = gaz.SaveWithActs(context.Background(), g, nil)
	pages := &onePage{}

	got, err := NewReadPage(gaz, memStorage{}, pages).Execute(context.Background(), g.ID, 3)

	if err != nil || got.Text != "PDF página" || got.Pages != 4 || got.Page != 3 || got.Gazette.Checksum != "abc" {
		t.Fatalf("veio %+v %v", got, err)
	}
	if pages.gotSource != domain.SourceDiarioCamara || pages.gotPage != 3 {
		t.Fatalf("a fonte e a página chegam ao extrator: %+v", pages)
	}
}

func TestReadPageRejectsPageZeroAndUnknownGazette(t *testing.T) {
	uc := NewReadPage(newMemGazettes(), memStorage{}, &onePage{})

	if _, err := uc.Execute(context.Background(), "x", 0); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("página 0: %v", err)
	}
	if _, err := uc.Execute(context.Background(), "nada", 1); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("edição desconhecida: %v", err)
	}
}
