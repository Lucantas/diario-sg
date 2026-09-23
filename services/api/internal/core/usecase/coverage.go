package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type SourceCoverage struct{ reader ports.CoverageReader }

func NewSourceCoverage(r ports.CoverageReader) *SourceCoverage { return &SourceCoverage{reader: r} }

func (uc *SourceCoverage) Execute(ctx context.Context) (domain.Coverage, error) {
	return uc.reader.Coverage(ctx)
}
