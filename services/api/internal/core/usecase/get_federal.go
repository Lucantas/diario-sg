package usecase

import (
	"context"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type FederalReport struct {
	TransfersFrom *time.Time
	TransfersTo   *time.Time
	Transfers     []domain.TransferTotal
	Amendments    []domain.Amendment
	Favored       []domain.FavoredTotal
	Special       []domain.SpecialTransfer
}

type GetFederal struct {
	reader  ports.FederalReader
	special ports.SpecialTransferReader
}

func NewGetFederal(reader ports.FederalReader, special ports.SpecialTransferReader) *GetFederal {
	return &GetFederal{reader: reader, special: special}
}

func (uc *GetFederal) Execute(ctx context.Context) (FederalReport, error) {
	var r FederalReport
	var err error
	if r.Transfers, err = uc.reader.TransferTotals(ctx); err != nil {
		return r, err
	}
	if r.TransfersFrom, r.TransfersTo, err = uc.reader.TransferMonths(ctx); err != nil {
		return r, err
	}
	if r.Amendments, err = uc.reader.Amendments(ctx); err != nil {
		return r, err
	}
	payments, err := uc.reader.AmendmentPayments(ctx)
	if err != nil {
		return r, err
	}
	r.Favored = domain.AggregateFavored(payments)
	r.Special, err = uc.special.SpecialTransfers(ctx)
	return r, err
}
