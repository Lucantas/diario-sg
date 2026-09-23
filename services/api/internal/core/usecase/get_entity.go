package usecase

import (
	"context"
	"fmt"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type GetEntity struct{ reader ports.EntityReader }

func NewGetEntity(r ports.EntityReader) *GetEntity { return &GetEntity{reader: r} }

func (uc *GetEntity) Execute(ctx context.Context, kind domain.EntityKind, input, source string) (domain.EntityReport, error) {
	if source != "" && !domain.ValidSource(source) {
		return domain.EntityReport{}, fmt.Errorf("%w: diário desconhecido %q", domain.ErrInvalidInput, source)
	}
	key, err := domain.ParseEntityInput(kind, input)
	if err != nil {
		return domain.EntityReport{}, err
	}
	return uc.reader.ReportByKey(ctx, kind, key, source)
}
