package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type GetOversight struct{ reader ports.OversightReader }

func NewGetOversight(reader ports.OversightReader) *GetOversight {
	return &GetOversight{reader: reader}
}

func (uc *GetOversight) Execute(ctx context.Context) (domain.TCEOversight, []domain.PenaltyProcess, error) {
	o, err := uc.reader.Oversight(ctx)
	if err != nil {
		return o, nil, err
	}
	return o, domain.GroupPenalties(o.Penalties), nil
}
