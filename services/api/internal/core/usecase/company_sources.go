package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type CompanySources struct {
	Registry  ports.RegistryReader
	Sanctions ports.SanctionReader
	Payments  ports.PaymentReader
	PNCP      ports.PNCPReader
	Works     ports.StalledWorkReader
	Federal   ports.AmendmentPaymentReader
	Municipal ports.MunicipalCommitmentReader
	Mural     ports.MuralReader
	Punished  ports.DiarioSanctionReader
}

func (s CompanySources) factsOf(ctx context.Context, cnpj string) (domain.CompanyFacts, error) {
	var f domain.CompanyFacts
	var err error
	if f.Registry, err = s.Registry.RegistryByCNPJ(ctx, cnpj); err != nil {
		return f, err
	}
	if f.RegistryMonth, err = s.Registry.RegistryMonth(ctx); err != nil {
		return f, err
	}
	if f.Sanctions, err = s.Sanctions.SanctionsByCNPJ(ctx, cnpj); err != nil {
		return f, err
	}
	if f.SanctionsListedOn, err = s.Sanctions.SanctionsListedOn(ctx); err != nil {
		return f, err
	}
	if f.Payments, err = s.Payments.PaymentsByCNPJ(ctx, cnpj); err != nil {
		return f, err
	}
	if f.PaymentsCoverage, err = s.Payments.PaymentsCoverage(ctx); err != nil {
		return f, err
	}
	if f.PNCPContracts, err = s.PNCP.PNCPContractsBySupplier(ctx, cnpj); err != nil {
		return f, err
	}
	if f.StalledWorks, err = s.Works.StalledWorksByCNPJ(ctx, cnpj); err != nil {
		return f, err
	}
	if f.AmendmentPayments, err = s.Federal.AmendmentPaymentsByCNPJ(ctx, cnpj); err != nil {
		return f, err
	}
	if f.Municipal, err = s.Municipal.MunicipalByCNPJ(ctx, cnpj); err != nil {
		return f, err
	}
	if f.Mural, err = s.Mural.MuralByCNPJ(ctx, cnpj); err != nil {
		return f, err
	}
	f.DiarioSanctions, err = s.diarioSanctions(ctx, cnpj)
	return f, err
}

func (s CompanySources) diarioSanctions(ctx context.Context, cnpj string) ([]domain.DiarioSanction, error) {
	candidates, err := s.Punished.SanctionCandidatesByCNPJ(ctx, cnpj)
	if err != nil {
		return nil, err
	}
	kinds := map[string]domain.DiarioSanctionKind{}
	var ids []string
	for _, c := range candidates {
		if kind, ok := domain.ClassifyDiarioSanction(c.Title, c.Body); ok {
			kinds[c.ActID] = kind
			ids = append(ids, c.ActID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	hits, err := s.Punished.HitsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.DiarioSanction, len(hits))
	for i, h := range hits {
		out[i] = domain.DiarioSanction{Kind: kinds[h.ID], Act: h}
	}
	return out, nil
}
