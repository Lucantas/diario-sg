package usecase

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestGetGazettePDFStreamsTheArchivedFile(t *testing.T) {
	gaz := newMemGazettes()
	g := &domain.Gazette{Checksum: "abc", StoragePath: "2026/09/18.pdf", PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)}
	_ = gaz.SaveWithActs(context.Background(), g, nil)
	uc := NewGetGazettePDF(gaz, memStorage{})

	got, err := uc.Gazette(context.Background(), g.ID)
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := uc.Open(context.Background(), got)
	if err != nil {
		t.Fatal(err)
	}
	defer pdf.Body.Close()
	b, _ := io.ReadAll(pdf.Body)

	if got.Checksum != "abc" || string(b) != "PDF" {
		t.Fatalf("veio %+v %q", got, b)
	}
}

func TestGetGazettePDFUnknownGazette(t *testing.T) {
	_, err := NewGetGazettePDF(newMemGazettes(), memStorage{}).Gazette(context.Background(), "nada")

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, veio %v", err)
	}
}

func TestGetGazettePDFRedirectsToSourceWhenNotArchived(t *testing.T) {
	g := domain.Gazette{ID: "g1", StoragePath: "1.pdf", SourceURL: "https://oficial/1.pdf"}

	pdf, err := NewGetGazettePDF(newMemGazettes(), failingStorage{}).Open(context.Background(), g)

	if err != nil || pdf.Body != nil || pdf.RedirectURL != g.SourceURL {
		t.Fatalf("%+v %v", pdf, err)
	}
}

func TestGetGazettePDFWithoutArchiveOrSourceFails(t *testing.T) {
	if _, err := NewGetGazettePDF(newMemGazettes(), failingStorage{}).Open(context.Background(), domain.Gazette{ID: "g1"}); err == nil {
		t.Fatal("sem arquivo e sem source_url deveria falhar")
	}
}
