package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type ExportActs struct{ acts ports.ActRepository }

func NewExportActs(a ports.ActRepository) *ExportActs { return &ExportActs{acts: a} }

func (uc *ExportActs) Execute(ctx context.Context, f domain.ActFilter, yield func(domain.ActHit, int) error) error {
	if err := f.Normalize(); err != nil {
		return err
	}
	f.Limit, f.Offset = domain.ExportLimit, 0
	return uc.acts.Export(ctx, f, yield)
}
