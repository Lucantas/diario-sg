package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const firstPortalYear = 2017

type LoadMunicipalCommitments struct {
	src  ports.MunicipalCommitmentSource
	repo ports.MunicipalCommitmentRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadMunicipalCommitments(src ports.MunicipalCommitmentSource, repo ports.MunicipalCommitmentRepository, runs ports.FetchRunRepository,
	raw ports.ObjectWriter, now func() time.Time) *LoadMunicipalCommitments {
	return &LoadMunicipalCommitments{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

func (uc *LoadMunicipalCommitments) Execute(ctx context.Context, from, to int) (domain.FetchRun, error) {
	started := uc.now()
	if from == 0 && to == 0 {
		from, to = started.Year()-1, started.Year()
	}
	if from < firstPortalYear || to < from || to > started.Year() {
		return domain.FetchRun{}, fmt.Errorf("%w: anos %d a %d (de %d até o ano corrente)", domain.ErrInvalidInput, from, to, firstPortalYear)
	}
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourceMunicipalCommitments, StartedAt: started,
		RequestedFrom: time.Date(from, 1, 1, 0, 0, 0, 0, time.UTC), RequestedTo: time.Date(to, 12, 31, 0, 0, 0, 0, time.UTC)}
	entities, err := uc.entities(ctx)
	for year := from; year <= to && err == nil; year++ {
		err = uc.loadYear(ctx, year, entities, &run)
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

func (uc *LoadMunicipalCommitments) entities(ctx context.Context) ([]domain.MunicipalEntity, error) {
	if err := uc.repo.Ready(ctx); err != nil {
		return nil, fmt.Errorf("tabelas dos empenhos do portal: %w", err)
	}
	body, err := uc.src.Entities(ctx)
	if err != nil {
		return nil, fmt.Errorf("entidades do portal: %w", err)
	}
	entities, err := domain.ParseMunicipalEntities(body)
	if err == nil && len(entities) == 0 {
		err = fmt.Errorf("o portal não listou nenhuma entidade")
	}
	return entities, err
}

func (uc *LoadMunicipalCommitments) loadYear(ctx context.Context, year int, entities []domain.MunicipalEntity, run *domain.FetchRun) error {
	var commitments []domain.MunicipalCommitment
	var totals []domain.MunicipalTotal
	for _, e := range entities {
		body, err := uc.src.Commitments(ctx, year, e.ID)
		if err != nil {
			return err
		}
		rows, total, err := domain.ParseMunicipalCommitments(body, year, e)
		if err != nil {
			return err
		}
		path := fmt.Sprintf("raw/%s/%s/%d-%d", domain.SourceMunicipalCommitments, uc.now().Format("2006/01/02"), year, e.ID)
		if err := archiveJSON(ctx, uc.raw, path, body, len(rows)); err != nil {
			return err
		}
		rows, repeated := domain.WithoutRepeatedCommitments(rows)
		run.Skipped += repeated
		commitments = append(commitments, rows...)
		if total != (domain.MunicipalTotal{Year: year, EntityID: e.ID, Entity: e.Name}) {
			totals = append(totals, total)
		}
	}
	if err := uc.repo.ReplaceMunicipalYear(ctx, year, commitments, totals); err != nil {
		return err
	}
	run.Found += len(commitments)
	run.Stored += len(commitments)
	return nil
}
