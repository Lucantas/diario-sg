package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type LoadFiscalTotals struct {
	src  ports.FiscalSource
	repo ports.FiscalRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadFiscalTotals(src ports.FiscalSource, repo ports.FiscalRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadFiscalTotals {
	return &LoadFiscalTotals{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

func (uc *LoadFiscalTotals) Execute(ctx context.Context) (domain.FetchRun, error) {
	started := uc.now()
	from, to := domain.FirstRREOYear, started.Year()
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourceSiconfi, StartedAt: started,
		RequestedFrom: time.Date(from, 1, 1, 0, 0, 0, 0, time.UTC), RequestedTo: time.Date(to, 12, 31, 0, 0, 0, 0, time.UTC)}
	totals, err := uc.collect(ctx, from, to)
	if err == nil {
		err = uc.repo.SaveFiscalTotals(ctx, totals)
	}
	if err == nil {
		run.Found, run.Stored = len(totals), len(totals)
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

func (uc *LoadFiscalTotals) collect(ctx context.Context, from, to int) ([]domain.FiscalTotal, error) {
	if err := uc.repo.Ready(ctx); err != nil {
		return nil, fmt.Errorf("tabela dos totais do SICONFI: %w", err)
	}
	var totals []domain.FiscalTotal
	for year := from; year <= to; year++ {
		total, ok, err := uc.latestPeriod(ctx, year)
		if err != nil {
			return nil, err
		}
		if ok {
			totals = append(totals, total)
		}
	}
	return totals, nil
}

func (uc *LoadFiscalTotals) latestPeriod(ctx context.Context, year int) (domain.FiscalTotal, bool, error) {
	for period := domain.LastRREOPeriod; period >= 1; period-- {
		body, err := uc.src.RREO(ctx, year, period)
		if err != nil {
			return domain.FiscalTotal{}, false, err
		}
		total, ok, err := domain.ParseRREOTotals(body, year, period)
		if err != nil {
			return domain.FiscalTotal{}, false, fmt.Errorf("RREO %d/%d: %w", period, year, err)
		}
		if !ok {
			continue
		}
		total.SourceURL = uc.src.RREOURL(year, period)
		path := fmt.Sprintf("raw/%s/%s/rreo_%d_%d", domain.SourceSiconfi, uc.now().Format("2006/01/02"), year, period)
		return total, true, archiveJSON(ctx, uc.raw, path, body, 1)
	}
	return domain.FiscalTotal{}, false, nil
}
