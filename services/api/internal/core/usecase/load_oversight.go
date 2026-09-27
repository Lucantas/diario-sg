package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const (
	datasetAccounts  = "prestacao_contas_municipio"
	datasetPenalties = "penalidades_ressarcimento_municipio"
	datasetWorks     = "obras_paralisadas"
)

type LoadOversight struct {
	src  ports.OversightSource
	repo ports.OversightRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadOversight(src ports.OversightSource, repo ports.OversightRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadOversight {
	return &LoadOversight{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

func (uc *LoadOversight) Execute(ctx context.Context) (domain.FetchRun, error) {
	started := uc.now()
	today := time.Date(started.Year(), started.Month(), started.Day(), 0, 0, 0, 0, time.UTC)
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourceOversight, StartedAt: started, RequestedFrom: today, RequestedTo: today}
	o, err := uc.collect(ctx)
	o = o.WithoutRepeats()
	if err == nil {
		err = uc.repo.ReplaceOversight(ctx, o)
	}
	if err == nil {
		run.Found = len(o.Accounts) + len(o.Penalties) + len(o.Works)
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

func (uc *LoadOversight) collect(ctx context.Context) (domain.TCEOversight, error) {
	var o domain.TCEOversight
	if err := uc.repo.Ready(ctx); err != nil {
		return o, fmt.Errorf("tabelas do controle do TCE-RJ: %w", err)
	}
	var err error
	if o.Accounts, err = fetchDataset(ctx, uc, datasetAccounts, domain.ParseTCEAccounts); err != nil {
		return o, err
	}
	if o.Penalties, err = fetchDataset(ctx, uc, datasetPenalties, domain.ParseTCEPenalties); err != nil {
		return o, err
	}
	o.Works, err = fetchDataset(ctx, uc, datasetWorks, domain.ParseStalledWorks)
	return o, err
}

func fetchDataset[T any](ctx context.Context, uc *LoadOversight, name string, parse func([]byte) ([]T, error)) ([]T, error) {
	body, err := uc.src.Dataset(ctx, name)
	if err != nil {
		return nil, err
	}
	rows, err := parse(body)
	if err != nil {
		return nil, err
	}
	if err := uc.archive(ctx, name, body, len(rows)); err != nil {
		return nil, err
	}
	return rows, nil
}

func (uc *LoadOversight) archive(ctx context.Context, name string, body []byte, rows int) error {
	dir := fmt.Sprintf("raw/%s/%s/", domain.SourceOversight, uc.now().Format("2006/01/02"))
	return archiveJSON(ctx, uc.raw, dir+name, body, rows)
}
