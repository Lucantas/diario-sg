package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type LoadMural struct {
	src  ports.MuralSource
	repo ports.MuralRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadMural(src ports.MuralSource, repo ports.MuralRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadMural {
	return &LoadMural{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

func (uc *LoadMural) Execute(ctx context.Context) (domain.FetchRun, error) {
	started := uc.now()
	today := time.Date(started.Year(), started.Month(), started.Day(), 0, 0, 0, 0, time.UTC)
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourceMural, StartedAt: started, RequestedFrom: today, RequestedTo: today}
	procurements, contracts, err := uc.collect(ctx)
	if err == nil {
		err = uc.repo.ReplaceMural(ctx, procurements, contracts)
	}
	if err == nil {
		run.Found = len(procurements) + len(contracts)
		run.Stored = run.Found
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

func (uc *LoadMural) collect(ctx context.Context) ([]domain.Procurement, []domain.ProcurementContract, error) {
	if err := uc.repo.Ready(ctx); err != nil {
		return nil, nil, fmt.Errorf("tabelas do mural: %w", err)
	}
	var procurements []domain.Procurement
	for _, list := range []string{domain.MuralTenders, domain.MuralDirect, domain.MuralUnenforceable} {
		page, err := uc.src.List(ctx, list)
		if err != nil {
			return nil, nil, err
		}
		rows, err := domain.ParseProcurements(list, page, uc.src.BaseURL())
		if err != nil {
			return nil, nil, err
		}
		if err := uc.archive(ctx, list, page, len(rows)); err != nil {
			return nil, nil, err
		}
		procurements = append(procurements, rows...)
	}
	page, err := uc.src.List(ctx, domain.MuralContracts)
	if err != nil {
		return nil, nil, err
	}
	contracts, err := domain.ParseProcurementContracts(page, uc.src.BaseURL())
	if err != nil {
		return nil, nil, err
	}
	return procurements, contracts, uc.archive(ctx, domain.MuralContracts, page, len(contracts))
}

func (uc *LoadMural) archive(ctx context.Context, list string, page []byte, rows int) error {
	path := fmt.Sprintf("raw/%s/%s/%s", domain.SourceMural, uc.now().Format("2006/01/02"), list)
	return archiveRaw(ctx, uc.raw, path, "html", page, rows)
}
