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
	// Reentrega da mesma mensagem: não duplica, mas republica o evento.
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
