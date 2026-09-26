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

type GetSupplierPanel struct {
	src      ports.PanelSource
	registry ports.RegistryReader
}

func NewGetSupplierPanel(src ports.PanelSource, registry ports.RegistryReader) *GetSupplierPanel {
	return &GetSupplierPanel{src: src, registry: registry}
}

func (uc *GetSupplierPanel) Execute(ctx context.Context, source string, f domain.PanelFilter) (domain.SupplierPanel, map[string]domain.ActHit, error) {
	source = domain.SourceOrDefault(source)
	if !domain.ValidSource(source) || (f.Year != 0 && (f.Year < firstPanelYear || f.Year > lastPanelYear)) {
		return domain.SupplierPanel{}, nil, domain.ErrInvalidFilter
	}
	acts, err := uc.src.PanelActs(ctx, source)
	if err != nil {
		return domain.SupplierPanel{}, nil, err
	}
	panel, err := uc.named(ctx, domain.BuildSupplierPanel(acts, f))
	if err != nil {
		return domain.SupplierPanel{}, nil, err
	}
	ids := make([]string, 0, len(panel.Rows))
	for _, row := range panel.Rows {
		if row.LargestActID != "" {
			ids = append(ids, row.LargestActID)
		}
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

func (uc *GetSupplierPanel) named(ctx context.Context, panel domain.SupplierPanel) (domain.SupplierPanel, error) {
	cnpjs := make([]string, len(panel.Rows))
	for i, row := range panel.Rows {
		cnpjs[i] = row.CNPJ
	}
	names, err := uc.registry.NamesByCNPJ(ctx, cnpjs)
	if err != nil {
		return panel, err
	}
	rows := make([]domain.SupplierRow, len(panel.Rows))
	for i, row := range panel.Rows {
		row.Name = names[row.CNPJ]
		rows[i] = row
	}
	panel.Rows = rows
	return panel, nil
}
