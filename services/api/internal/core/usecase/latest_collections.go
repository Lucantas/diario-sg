package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type CollectorRun struct {
	Collector domain.Collector
	Run       domain.FetchRun
}

type LatestCollections struct{ runs ports.LatestRunsReader }

func NewLatestCollections(runs ports.LatestRunsReader) *LatestCollections {
	return &LatestCollections{runs: runs}
}

func (uc *LatestCollections) Execute(ctx context.Context) ([]CollectorRun, error) {
	latest, err := uc.runs.LatestRuns(ctx, domain.CollectorSources())
	if err != nil {
		return nil, err
	}
	var out []CollectorRun
	for _, c := range domain.Collectors {
		if run, ok := latest[c.Source]; ok {
			out = append(out, CollectorRun{Collector: c, Run: run})
		}
	}
	return out, nil
}
