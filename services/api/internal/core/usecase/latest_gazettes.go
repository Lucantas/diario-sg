package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type LatestGazettes struct{ days ports.LatestDaysReader }

func NewLatestGazettes(d ports.LatestDaysReader) *LatestGazettes { return &LatestGazettes{days: d} }

func (uc *LatestGazettes) Execute(ctx context.Context) ([]domain.LatestDay, error) {
	days, err := uc.days.LatestDays(ctx)
	if err != nil {
		return nil, err
	}
	domain.SortLatestDays(days)
	return days, nil
}
