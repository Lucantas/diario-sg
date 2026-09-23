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
	uc := NewIndexGazette(repo, memStorage{}, fixedExtractor{"PORTARIA 1\nDECRETO 2 CNPJ"}, lineParsers{}, cnpjExtractor{}, pub)

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
	uc := NewIndexGazette(newMemGazettes(), memStorage{}, fixedExtractor{}, lineParsers{}, cnpjExtractor{}, &recPublisher{})
	err := uc.Execute(context.Background(), IndexGazetteInput{})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("esperava ErrInvalidInput, veio %v", err)
	}
}

func TestIndexGazette_EditionNumberComesFromTextWhenEventHasNone(t *testing.T) {
	repo := newMemGazettes()
	uc := NewIndexGazette(repo, memStorage{}, fixedExtractor{"EDIÇÃO 1771\nDECRETO 1"}, lineParsers{}, cnpjExtractor{}, &recPublisher{})
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

func TestIndexGazette_UsesTheParserOfTheSource(t *testing.T) {
	repo := newMemGazettes()
	uc := NewIndexGazette(repo, memStorage{}, fixedExtractor{"PORTARIA 1"}, lineParsers{}, cnpjExtractor{}, &recPublisher{})
	base := IndexGazetteInput{PublishedAt: time.Now(), StoragePath: "x.pdf"}

	camara := base
	camara.Checksum, camara.Source = "camara", domain.SourceDiarioCamara
	prefeitura := base
	prefeitura.Checksum = "prefeitura"
	for _, in := range []IndexGazetteInput{camara, prefeitura} {
		if err := uc.Execute(context.Background(), in); err != nil {
			t.Fatal(err)
		}
	}

	if g := repo.saved["g-camara"]; g.Source != domain.SourceDiarioCamara || repo.acts["g-camara"][0].Title != "câmara: PORTARIA 1" {
		t.Errorf("edição da Câmara: fonte %q, ato %q", g.Source, repo.acts["g-camara"][0].Title)
	}
	if g := repo.saved["g-prefeitura"]; g.Source != domain.SourceDiarioPrefeitura || repo.acts["g-prefeitura"][0].Title != "PORTARIA 1" {
		t.Errorf("sem fonte deveria ser a Prefeitura: fonte %q, ato %q", g.Source, repo.acts["g-prefeitura"][0].Title)
	}
}

func TestIndexGazette_UnknownSourceIsPermanent(t *testing.T) {
	uc := NewIndexGazette(newMemGazettes(), memStorage{}, fixedExtractor{}, lineParsers{}, cnpjExtractor{}, &recPublisher{})

	err := uc.Execute(context.Background(), IndexGazetteInput{PublishedAt: time.Now(), StoragePath: "x.pdf", Checksum: "c", Source: "tce"})

	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("esperava ErrInvalidInput, veio %v", err)
	}
}
