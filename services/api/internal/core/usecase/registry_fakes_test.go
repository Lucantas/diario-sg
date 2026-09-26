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
