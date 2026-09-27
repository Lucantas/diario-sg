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
	f.Municipal, err = s.Municipal.MunicipalByCNPJ(ctx, cnpj)
	return f, err
}
