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
	f.PaymentsCoverage, err = s.Payments.PaymentsCoverage(ctx)
	return f, err
}
