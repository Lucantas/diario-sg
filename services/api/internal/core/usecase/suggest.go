package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type Suggest struct{ source ports.SuggestionSource }

func NewSuggest(s ports.SuggestionSource) *Suggest { return &Suggest{source: s} }

func (uc *Suggest) Execute(ctx context.Context, q string) ([]domain.Suggestion, error) {
	l := domain.ParseLookup(q)
	out := []domain.Suggestion{}
	if l.CNPJ != "" {
		company, found, err := uc.source.CompanyByCNPJ(ctx, l.CNPJ)
		if err != nil {
			return nil, err
		}
		if found {
			out = append(out, company)
		}
	}
	for _, n := range []struct {
		kind       domain.EntityKind
		key, label string
	}{{domain.EntityProcesso, l.Processo, l.ProcessoLabel}, {domain.EntityContrato, l.Contrato, l.ContratoLabel}} {
		if n.key == "" {
			continue
		}
		acts, err := uc.source.EntityActs(ctx, n.kind, n.key)
		if err != nil {
			return nil, err
		}
		if acts > 0 {
			out = append(out, domain.Suggestion{Kind: n.kind, Key: n.key, Label: n.label, Acts: acts})
		}
	}
	if l.Name != "" {
		byName, err := uc.source.CompaniesByName(ctx, l.Name, domain.SuggestNameLimit)
		if err != nil {
			return nil, err
		}
		out = append(out, byName...)
	}
	return out, nil
}
