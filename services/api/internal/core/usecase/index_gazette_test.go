package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestIndexGazette_SavesActsAndIsIdempotent(t *testing.T) {
	repo := newMemGazettes()
	pub := &recPublisher{}
	uc := NewIndexGazette(repo, memStorage{}, fixedExtractor{"PORTARIA 1\nDECRETO 2 CNPJ"}, lineParser{}, cnpjExtractor{}, pub)

	in := IndexGazetteInput{PublishedAt: time.Now(), StoragePath: "gazettes/x.pdf", Checksum: "abc"}
	if err := uc.Execute(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if got := len(repo.acts["g-abc"]); got != 2 {
		t.Fatalf("esperava 2 atos, veio %d", got)
	}
	if saved := repo.acts["g-abc"]; len(saved[0].Entities) != 0 || len(saved[1].Entities) != 1 || saved[1].Entities[0].Normalized != "12345678000190" {
		t.Fatalf("entidades devem ser extraídas por ato: %+v", saved)
	}

	if err := uc.Execute(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if len(repo.saved) != 1 || len(pub.indexed) != 2 {
		t.Errorf("idempotência quebrada: saved=%d eventos=%d", len(repo.saved), len(pub.indexed))
	}
}

func TestIndexGazette_InvalidInputIsPermanent(t *testing.T) {
	uc := NewIndexGazette(newMemGazettes(), memStorage{}, fixedExtractor{}, lineParser{}, cnpjExtractor{}, &recPublisher{})
	err := uc.Execute(context.Background(), IndexGazetteInput{})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("esperava ErrInvalidInput, veio %v", err)
	}
}

func TestIndexGazette_EditionNumberComesFromTextWhenEventHasNone(t *testing.T) {
	repo := newMemGazettes()
	uc := NewIndexGazette(repo, memStorage{}, fixedExtractor{"EDIÇÃO 1771\nDECRETO 1"}, lineParser{}, cnpjExtractor{}, &recPublisher{})
	base := IndexGazetteInput{PublishedAt: time.Now(), StoragePath: "x.pdf"}

	in := base
	in.Checksum = "sem-numero"
	if err := uc.Execute(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if got := repo.saved["g-sem-numero"].EditionNumber; got != "1771" {
		t.Errorf("esperava número lido do texto, veio %q", got)
	}

	in = base
	in.Checksum, in.EditionNumber = "com-numero", "42"
	if err := uc.Execute(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if got := repo.saved["g-com-numero"].EditionNumber; got != "42" {
		t.Errorf("número do evento deve prevalecer, veio %q", got)
	}
}
