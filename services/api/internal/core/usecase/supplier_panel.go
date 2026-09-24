package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const (
	firstPanelYear = 2000
	lastPanelYear  = 2100
)

type GetSupplierPanel struct{ src ports.PanelSource }

func NewGetSupplierPanel(src ports.PanelSource) *GetSupplierPanel { return &GetSupplierPanel{src: src} }

func (uc *GetSupplierPanel) Execute(ctx context.Context, source string, f domain.PanelFilter) (domain.SupplierPanel, map[string]domain.ActHit, error) {
	source = domain.SourceOrDefault(source)
	if !domain.ValidSource(source) || (f.Year != 0 && (f.Year < firstPanelYear || f.Year > lastPanelYear)) {
		return domain.SupplierPanel{}, nil, domain.ErrInvalidFilter
	}
	acts, err := uc.src.PanelActs(ctx, source)
	if err != nil {
		return domain.SupplierPanel{}, nil, err
	}
	panel := domain.BuildSupplierPanel(acts, f)
	ids := make([]string, 0, len(panel.Rows))
	for _, row := range panel.Rows {
		ids = append(ids, row.LargestActID)
	}
	largest := map[string]domain.ActHit{}
	if len(ids) == 0 {
		return panel, largest, nil
	}
	hits, err := uc.src.HitsByIDs(ctx, ids)
	if err != nil {
		return domain.SupplierPanel{}, nil, err
	}
	for _, h := range hits {
		largest[h.ID] = h
	}
	return panel, largest, nil
}
