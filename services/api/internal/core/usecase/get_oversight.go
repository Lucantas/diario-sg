package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type OversightView struct {
	domain.TCEOversight
	Processes []domain.PenaltyProcess
	Fiscal    []domain.FiscalControl
}

type GetOversight struct {
	reader ports.OversightReader
	fiscal ports.FiscalReader
}

func NewGetOversight(reader ports.OversightReader, fiscal ports.FiscalReader) *GetOversight {
	return &GetOversight{reader: reader, fiscal: fiscal}
}

func (uc *GetOversight) Execute(ctx context.Context) (OversightView, error) {
	o, err := uc.reader.Oversight(ctx)
	if err != nil {
		return OversightView{}, err
	}
	totals, err := uc.fiscal.FiscalTotals(ctx)
	if err != nil {
		return OversightView{}, err
	}
	paid, err := uc.fiscal.PaidByYear(ctx)
	if err != nil {
		return OversightView{}, err
	}
	return OversightView{TCEOversight: o, Processes: domain.GroupPenalties(o.Penalties), Fiscal: domain.BuildFiscalControl(totals, paid)}, nil
}
