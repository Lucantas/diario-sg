package usecase

import (
	"context"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeRegistry struct {
	byCNPJ    map[string]*domain.CompanyRegistry
	month     *time.Time
	asked     []string
	sanctions map[string][]domain.Sanction
	payments  map[string][]domain.PaymentYear
	pncp      map[string][]domain.PNCPContract
	paid      []domain.PaidTotal
}

func (f *fakeRegistry) SanctionsByCNPJ(_ context.Context, cnpj string) ([]domain.Sanction, error) {
	return f.sanctions[cnpj], nil
}

func (f *fakeRegistry) SanctionsListedOn(context.Context) (map[string]time.Time, error) {
	return map[string]time.Time{domain.RegisterCEIS: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)}, nil
}

func (f *fakeRegistry) RegistryByCNPJ(_ context.Context, cnpj string) (*domain.CompanyRegistry, error) {
	f.asked = append(f.asked, cnpj)
	return f.byCNPJ[cnpj], nil
}

func (f *fakeRegistry) RegistryMonth(context.Context) (*time.Time, error) { return f.month, nil }

func (f *fakeRegistry) NamesByCNPJ(_ context.Context, cnpjs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, c := range cnpjs {
		if r := f.byCNPJ[c]; r != nil {
			out[c] = r.Company.Name
		}
	}
	return out, nil
}

func (f *fakeRegistry) PaymentsByCNPJ(_ context.Context, cnpj string) ([]domain.PaymentYear, error) {
	return f.payments[cnpj], nil
}

func (f *fakeRegistry) PaymentsCoverage(context.Context) (*domain.PaymentCoverage, error) {
	return nil, nil
}

func (f *fakeRegistry) PaidByCNPJYear(context.Context) ([]domain.PaidTotal, error) {
	return f.paid, nil
}

func (f *fakeRegistry) PNCPContractsBySupplier(_ context.Context, cnpj string) ([]domain.PNCPContract, error) {
	return f.pncp[cnpj], nil
}

func (f *fakeRegistry) StalledWorksByCNPJ(context.Context, string) ([]domain.StalledWork, error) {
	return nil, nil
}

func fakeSources(f *fakeRegistry) CompanySources {
	return CompanySources{Registry: f, Sanctions: f, Payments: f, PNCP: f, Works: f}
}
