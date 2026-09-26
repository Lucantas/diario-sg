package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type LoadPNCP struct {
	src  ports.PNCPSource
	repo ports.PNCPRepository
	runs ports.FetchRunRepository
	raw  ports.ObjectWriter
	now  func() time.Time
}

func NewLoadPNCP(src ports.PNCPSource, repo ports.PNCPRepository, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time) *LoadPNCP {
	return &LoadPNCP{src: src, repo: repo, runs: runs, raw: raw, now: now}
}

func (uc *LoadPNCP) Execute(ctx context.Context, from, to int) (domain.FetchRun, error) {
	started := uc.now()
	if from == 0 && to == 0 {
		from, to = domain.FirstPNCPYear, started.Year()
	}
	if from < domain.FirstPNCPYear || to < from || to > started.Year() {
		return domain.FetchRun{}, fmt.Errorf("%w: anos %d a %d (de %d até o ano corrente)", domain.ErrInvalidInput, from, to, domain.FirstPNCPYear)
	}
	run := domain.FetchRun{ID: domain.NewRunID(), Source: domain.SourcePNCP, StartedAt: started,
		RequestedFrom: time.Date(from, 1, 1, 0, 0, 0, 0, time.UTC), RequestedTo: time.Date(to, 12, 31, 0, 0, 0, 0, time.UTC)}
	contracts, manifest, err := uc.collect(ctx, from, to, &run)
	if err == nil {
		err = uc.archive(ctx, contracts, manifest)
	}
	if err == nil {
		err = uc.repo.ReplaceYears(ctx, from, to, contracts)
	}
	if err == nil {
		run.Stored = len(contracts)
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

func (uc *LoadPNCP) collect(ctx context.Context, from, to int, run *domain.FetchRun) ([]domain.PNCPContract, map[string]string, error) {
	if err := uc.repo.Ready(ctx); err != nil {
		return nil, nil, fmt.Errorf("tabela do PNCP: %w", err)
	}
	var out []domain.PNCPContract
	manifest := map[string]string{}
	for _, org := range domain.MunicipalOrgCNPJs() {
		for year := from; year <= to; year++ {
			sum, err := uc.src.Contracts(ctx, org, year, func(c domain.PNCPContract, company bool) error {
				run.Found++
				if !company {
					run.Skipped++
					return nil
				}
				out = append(out, c)
				return nil
			})
			if err != nil {
				return nil, nil, err
			}
			manifest[fmt.Sprintf("%s/%d", org, year)] = sum
		}
	}
	return out, manifest, nil
}

func (uc *LoadPNCP) archive(ctx context.Context, contracts []domain.PNCPContract, manifest map[string]string) error {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	enc := json.NewEncoder(gz)
	for _, c := range contracts {
		if err := enc.Encode(c); err != nil {
			return err
		}
	}
	if err := gz.Close(); err != nil {
		return err
	}
	dir := fmt.Sprintf("raw/%s/%s/", domain.SourcePNCP, uc.now().Format("2006/01/02"))
	if err := uc.raw.Put(ctx, dir+"contratos.jsonl.gz", "application/gzip", &buf); err != nil {
		return err
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return uc.raw.Put(ctx, dir+"manifest.json", "application/json", bytes.NewReader(body))
}
