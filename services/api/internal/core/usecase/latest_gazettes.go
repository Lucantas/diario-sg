package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type LatestGazettes struct{ editions ports.LatestEditionsReader }

func NewLatestGazettes(e ports.LatestEditionsReader) *LatestGazettes {
	return &LatestGazettes{editions: e}
}

func (uc *LatestGazettes) Execute(ctx context.Context) ([]domain.LatestEdition, error) {
	editions, err := uc.editions.LatestEditions(ctx)
	if err != nil {
		return nil, err
	}
	domain.SortLatestEditions(editions)
	return editions, nil
}
