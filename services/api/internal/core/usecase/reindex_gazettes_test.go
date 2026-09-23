package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type pathStorage map[string]string

func (s pathStorage) Get(_ context.Context, path string) (io.ReadCloser, error) {
	text, ok := s[path]
	if !ok {
		return nil, errors.New("objeto não encontrado")
	}
	return io.NopCloser(strings.NewReader(text)), nil
}

type echoExtractor struct{}

func (echoExtractor) Extract(_ context.Context, r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	return string(b), err
}

func day(d int) time.Time { return time.Date(2026, 9, d, 0, 0, 0, 0, time.UTC) }

func seed(repo *memGazettes, id string, published time.Time, number string, acts ...domain.Act) {
	repo.saved[id] = domain.Gazette{ID: id, EditionNumber: number, PublishedAt: published, StoragePath: id + ".pdf"}
	repo.acts[id] = acts
}

func TestReindexGazettes_ReplacesActsInThePeriodWithoutPublishing(t *testing.T) {
	repo := newMemGazettes()
	seed(repo, "a", day(10), "1", domain.Act{Title: "antigo"})
	seed(repo, "b", day(11), "", domain.Act{Title: "antigo"})
	seed(repo, "fora", day(20), "9", domain.Act{Title: "antigo"})
	storage := pathStorage{"a.pdf": "PORTARIA 1\nDECRETO 2 CNPJ", "b.pdf": "EDIÇÃO 1771\nPORTARIA 3", "fora.pdf": "X"}
	uc := NewReindexGazettes(repo, storage, echoExtractor{}, lineParsers{}, cnpjExtractor{})

	res, err := uc.Execute(context.Background(), day(10), day(11))

	if err != nil {
		t.Fatal(err)
	}
	if res != (ReindexResult{Found: 2, Reindexed: 2}) {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if got := repo.acts["a"]; len(got) != 2 || got[0].Title != "PORTARIA 1" || len(got[1].Entities) != 1 {
		t.Fatalf("atos de a não foram trocados com entidades: %+v", got)
	}
	if repo.saved["a"].EditionNumber != "1" || repo.saved["b"].EditionNumber != "1771" {
		t.Fatalf("número: sem número no texto mantém o antigo, com número atualiza: a=%q b=%q",
			repo.saved["a"].EditionNumber, repo.saved["b"].EditionNumber)
	}
	if repo.acts["fora"][0].Title != "antigo" {
		t.Fatal("edição fora do período não pode ser tocada")
	}
}

func TestReindexGazettes_FailureKeepsOldActsAndContinues(t *testing.T) {
	repo := newMemGazettes()
	seed(repo, "sem-pdf", day(10), "1", domain.Act{Title: "antigo"})
	seed(repo, "ok", day(11), "2", domain.Act{Title: "antigo"})
	uc := NewReindexGazettes(repo, pathStorage{"ok.pdf": "PORTARIA 1"}, echoExtractor{}, lineParsers{}, cnpjExtractor{})

	res, err := uc.Execute(context.Background(), day(10), day(11))

	if err == nil || !strings.Contains(err.Error(), "sem-pdf.pdf") {
		t.Fatalf("esperava erro citando a edição que falhou, veio %v", err)
	}
	if res != (ReindexResult{Found: 2, Reindexed: 1, Failed: 1}) {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if repo.acts["sem-pdf"][0].Title != "antigo" || repo.acts["ok"][0].Title != "PORTARIA 1" {
		t.Fatalf("falha não pode apagar atos nem parar as demais: %+v", repo.acts)
	}
}

func TestReindexGazettes_StopsWhenContextIsCancelled(t *testing.T) {
	repo := newMemGazettes()
	seed(repo, "a", day(10), "1", domain.Act{Title: "antigo"})
	seed(repo, "b", day(11), "2", domain.Act{Title: "antigo"})
	storage := pathStorage{"a.pdf": "PORTARIA 1", "b.pdf": "PORTARIA 2"}
	uc := NewReindexGazettes(repo, storage, echoExtractor{}, lineParsers{}, cnpjExtractor{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, err := uc.Execute(ctx, day(10), day(11))

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("esperava context.Canceled, veio %v", err)
	}
	if res != (ReindexResult{Found: 2, Reindexed: 0, Failed: 0}) {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if repo.acts["a"][0].Title != "antigo" || repo.acts["b"][0].Title != "antigo" {
		t.Fatalf("contexto cancelado não pode tocar os atos: %+v", repo.acts)
	}
}

func TestReindexGazettes_RejectsInvertedPeriod(t *testing.T) {
	uc := NewReindexGazettes(newMemGazettes(), pathStorage{}, echoExtractor{}, lineParsers{}, cnpjExtractor{})

	_, err := uc.Execute(context.Background(), day(11), day(10))

	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("esperava ErrInvalidInput, veio %v", err)
	}
}

func TestReindexGazettes_UsesTheParserOfTheGazetteSource(t *testing.T) {
	repo := newMemGazettes()
	seed(repo, "c", day(10), "7")
	g := repo.saved["c"]
	g.Source = domain.SourceDiarioCamara
	repo.saved["c"] = g
	uc := NewReindexGazettes(repo, pathStorage{"c.pdf": "PORTARIA 1"}, echoExtractor{}, lineParsers{}, cnpjExtractor{})

	if _, err := uc.Execute(context.Background(), day(10), day(10)); err != nil {
		t.Fatal(err)
	}

	if got := repo.acts["c"]; len(got) != 1 || got[0].Title != "câmara: PORTARIA 1" {
		t.Errorf("deveria usar o parser da Câmara: %+v", got)
	}
}
