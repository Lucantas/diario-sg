package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type ReadAct struct {
	gazettes ports.GazetteRepository
	acts     ports.ActRepository
}

func NewReadAct(g ports.GazetteRepository, a ports.ActRepository) *ReadAct {
	return &ReadAct{gazettes: g, acts: a}
}

func (uc *ReadAct) Execute(ctx context.Context, gazetteID string, position int) (domain.Gazette, domain.Act, error) {
	g, err := uc.gazettes.FindByID(ctx, gazetteID)
	if err != nil {
		return domain.Gazette{}, domain.Act{}, err
	}
	acts, err := uc.acts.ListByGazette(ctx, gazetteID)
	if err != nil {
		return domain.Gazette{}, domain.Act{}, err
	}
	for _, a := range acts {
		if a.Position == position {
			return g, a, nil
		}
	}
	return domain.Gazette{}, domain.Act{}, domain.ErrNotFound
}
