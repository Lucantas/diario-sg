package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeLatestEditions []domain.LatestEdition

func (f fakeLatestEditions) LatestEditions(context.Context) ([]domain.LatestEdition, error) {
	return f, nil
}

func TestLatestGazettesSortsEditionsAndTypes(t *testing.T) {
	day := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	reader := fakeLatestEditions{
		{Source: domain.SourceDiarioCamara, Day: day},
		{Source: domain.SourceDiarioPrefeitura, Day: day, Types: []domain.TypeTotal{{Type: domain.ActDecreto, Acts: 1}, {Type: domain.ActNomeacao, Acts: 9}}},
	}

	got, err := NewLatestGazettes(reader).Execute(context.Background())

	if err != nil || got[0].Source != domain.SourceDiarioPrefeitura || got[0].Types[0].Type != domain.ActNomeacao {
		t.Fatalf("Prefeitura primeiro e tipo mais frequente primeiro: %+v %v", got, err)
	}
}
