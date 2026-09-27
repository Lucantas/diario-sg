package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type LoadNorms struct {
	src  ports.NormSource
	repo ports.NormRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadNorms(src ports.NormSource, repo ports.NormRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadNorms {
	return &LoadNorms{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

func (uc *LoadNorms) Execute(ctx context.Context) (domain.FetchRun, error) {
	started := uc.now()
	today := time.Date(started.Year(), started.Month(), started.Day(), 0, 0, 0, 0, time.UTC)
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourceNorms, StartedAt: started, RequestedFrom: today, RequestedTo: today}
	norms, err := uc.collect(ctx, &run)
	if err == nil {
		err = uc.repo.ReplaceNorms(ctx, norms)
	}
	if err == nil {
		run.Stored = len(norms)
	}
	run.FinishedAt = uc.now()
	if err != nil {
		run.Error = err.Error()
	}
	if saveErr := uc.runs.Save(ctx, run); saveErr != nil && err == nil {
		err = saveErr
	}
	return run, err
}

func (uc *LoadNorms) collect(ctx context.Context, run *domain.FetchRun) ([]domain.Norm, error) {
	if err := uc.repo.Ready(ctx); err != nil {
		return nil, fmt.Errorf("tabela das normas: %w", err)
	}
	var all []domain.Norm
	for _, kind := range domain.NormKindsInOrder {
		page, err := uc.src.Norms(ctx, domain.NormCategories[kind])
		if err != nil {
			return nil, err
		}
		norms, invalid, err := domain.ParseNorms(kind, page, uc.src.BaseURL())
		if err != nil {
			return nil, err
		}
		path := fmt.Sprintf("raw/%s/%s/%s", domain.SourceNorms, uc.now().Format("2006/01/02"), kind)
		if err := archiveRaw(ctx, uc.raw, path, "html", page, len(norms)); err != nil {
			return nil, err
		}
		run.Found += len(norms) + invalid
		run.Failed += invalid
		all = append(all, norms...)
	}
	all, repeated := domain.WithoutRepeatedNorms(all)
	run.Skipped += repeated
	return all, nil
}
