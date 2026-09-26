package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type GetEntity struct {
	reader    ports.EntityReader
	registry  ports.RegistryReader
	sanctions ports.SanctionReader
}

func NewGetEntity(r ports.EntityReader, registry ports.RegistryReader, sanctions ports.SanctionReader) *GetEntity {
	return &GetEntity{reader: r, registry: registry, sanctions: sanctions}
}

func (uc *GetEntity) Execute(ctx context.Context, kind domain.EntityKind, input, source string) (domain.EntityReport, error) {
	if source != "" && !domain.ValidSource(source) {
		return domain.EntityReport{}, fmt.Errorf("%w: diário desconhecido %q", domain.ErrInvalidInput, source)
	}
	key, err := domain.ParseEntityInput(kind, input)
	if err != nil {
		return domain.EntityReport{}, err
	}
	report, err := uc.reader.ReportByKey(ctx, kind, key, source)
	if err != nil {
		return report, err
	}
	if kind == domain.EntityCNPJ {
		if report.Registry, report.RegistryMonth, err = registryOf(ctx, uc.registry, key); err != nil {
			return report, err
		}
		if report.Sanctions, report.SanctionsListedOn, err = sanctionsOf(ctx, uc.sanctions, key); err != nil {
			return report, err
		}
	}
	return withPhases(report), nil
}

func registryOf(ctx context.Context, r ports.RegistryReader, cnpj string) (*domain.CompanyRegistry, *time.Time, error) {
	reg, err := r.RegistryByCNPJ(ctx, cnpj)
	if err != nil {
		return nil, nil, err
	}
	month, err := r.RegistryMonth(ctx)
	return reg, month, err
}

func sanctionsOf(ctx context.Context, r ports.SanctionReader, cnpj string) ([]domain.Sanction, map[string]time.Time, error) {
	sanctions, err := r.SanctionsByCNPJ(ctx, cnpj)
	if err != nil {
		return nil, nil, err
	}
	listed, err := r.SanctionsListedOn(ctx)
	return sanctions, listed, err
}

func withPhases(r domain.EntityReport) domain.EntityReport {
	acts := make([]domain.ActHit, len(r.Acts))
	for i, h := range r.Acts {
		h.Phase = domain.PhaseOf(h.Type, h.Title)
		acts[i] = h
	}
	r.Acts = acts
	r.CountByPhase = map[domain.Phase]int{}
	for _, c := range r.TypeTitleCounts {
		r.CountByPhase[domain.PhaseOf(c.Type, c.Title)] += c.Acts
	}
	return r
}
