package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type recFetchRuns struct{ saved []domain.FetchRun }

func (r *recFetchRuns) Save(_ context.Context, run domain.FetchRun) error {
	r.saved = append(r.saved, run)
	return nil
}

func validRun() domain.FetchRun {
	start := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	return domain.FetchRun{ID: "0b5f3f7e-1c2d-4e5f-8a9b-0c1d2e3f4a5b", Source: domain.SourceDiarioPrefeitura,
		RequestedFrom: start.AddDate(0, 0, -3), RequestedTo: start, Found: 2, Stored: 1, Skipped: 1,
		StartedAt: start, FinishedAt: start.Add(time.Minute)}
}

func TestRecordFetchRunSavesValidRuns(t *testing.T) {
	repo := &recFetchRuns{}

	if err := NewRecordFetchRun(repo).Execute(context.Background(), validRun()); err != nil || len(repo.saved) != 1 {
		t.Fatalf("coleta válida não gravada: %v %+v", err, repo.saved)
	}
}

func TestRecordFetchRunRejectsInconsistentRuns(t *testing.T) {
	repo := &recFetchRuns{}
	uc := NewRecordFetchRun(repo)
	broken := []func(*domain.FetchRun){
		func(r *domain.FetchRun) { r.ID = "" },
		func(r *domain.FetchRun) { r.Source = "" },
		func(r *domain.FetchRun) { r.Failed = -1 },
		func(r *domain.FetchRun) { r.FinishedAt = r.StartedAt.Add(-time.Second) },
		func(r *domain.FetchRun) { r.RequestedTo = r.RequestedFrom.AddDate(0, 0, -1) },
	}
	for i, breakIt := range broken {
		run := validRun()
		breakIt(&run)
		if err := uc.Execute(context.Background(), run); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("caso %d: esperava ErrInvalidInput, veio %v", i, err)
		}
	}
	if len(repo.saved) != 0 {
		t.Fatal("coleta inconsistente não deveria ser gravada")
	}
}
