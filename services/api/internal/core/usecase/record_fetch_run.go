package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type RecordFetchRun struct{ repo ports.FetchRunRepository }

func NewRecordFetchRun(r ports.FetchRunRepository) *RecordFetchRun { return &RecordFetchRun{repo: r} }

func (uc *RecordFetchRun) Execute(ctx context.Context, run domain.FetchRun) error {
	if !run.Valid() {
		return domain.ErrInvalidInput
	}
	return uc.repo.Save(ctx, run)
}
