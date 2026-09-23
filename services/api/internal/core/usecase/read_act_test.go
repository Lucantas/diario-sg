package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type gazetteActs struct {
	ports.ActRepository
	acts []domain.Act
}

func (g gazetteActs) ListByGazette(context.Context, string) ([]domain.Act, error) { return g.acts, nil }

type fixedCoverage struct{ c domain.Coverage }

func (f fixedCoverage) Coverage(context.Context) (domain.Coverage, error) { return f.c, nil }

func TestReadActFindsTheActByPosition(t *testing.T) {
	ctx := context.Background()
	gazettes := newMemGazettes()
	g := domain.Gazette{Checksum: "abc", EditionNumber: "9"}
	_ = gazettes.SaveWithActs(ctx, &g, nil)
	acts := gazetteActs{acts: []domain.Act{{Position: 0, Title: "primeiro"}, {Position: 1, Title: "segundo"}}}
	uc := NewReadAct(gazettes, acts)

	got, act, err := uc.Execute(ctx, g.ID, 1)

	if err != nil || got.EditionNumber != "9" || act.Title != "segundo" {
		t.Fatalf("veio %+v %+v %v", got, act, err)
	}
	if _, _, err := uc.Execute(ctx, g.ID, 7); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("posição inexistente deveria ser ErrNotFound, veio %v", err)
	}
	if _, _, err := uc.Execute(ctx, "outra", 0); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("edição inexistente deveria ser ErrNotFound, veio %v", err)
	}
}

func TestSourceCoverageReturnsTheReaderCoverage(t *testing.T) {
	want := domain.Coverage{First: time.Date(2010, 1, 4, 0, 0, 0, 0, time.UTC), Gazettes: 3, Acts: 10}

	got, err := NewSourceCoverage(fixedCoverage{want}).Execute(context.Background())

	if err != nil || got != want {
		t.Fatalf("veio %+v %v", got, err)
	}
}
