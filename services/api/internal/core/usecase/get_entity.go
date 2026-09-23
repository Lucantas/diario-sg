package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type GetEntity struct{ reader ports.EntityReader }

func NewGetEntity(r ports.EntityReader) *GetEntity { return &GetEntity{reader: r} }

func (uc *GetEntity) Execute(ctx context.Context, kind domain.EntityKind, input string) (domain.EntityReport, error) {
	key, err := domain.ParseEntityInput(kind, input)
	if err != nil {
		return domain.EntityReport{}, err
	}
	return uc.reader.ReportByKey(ctx, kind, key)
}
