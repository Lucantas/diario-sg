package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeSuggestionSource struct {
	companies map[string]domain.Suggestion
	entities  map[string]int
	byName    []domain.Suggestion
	nameQuery string
	err       error
}

func (f *fakeSuggestionSource) CompanyByCNPJ(_ context.Context, cnpj string) (domain.Suggestion, bool, error) {
	s, ok := f.companies[cnpj]
	return s, ok, f.err
}

func (f *fakeSuggestionSource) EntityActs(_ context.Context, kind domain.EntityKind, key string) (int, error) {
	return f.entities[string(kind)+":"+key], f.err
}

func (f *fakeSuggestionSource) CompaniesByName(_ context.Context, text string, _ int) ([]domain.Suggestion, error) {
	f.nameQuery = text
	return f.byName, f.err
}

func TestSuggestFindsTheCompanyOfACNPJ(t *testing.T) {
	company := domain.Suggestion{Kind: domain.EntityCNPJ, Key: "12345678000190", Label: "12.345.678/0001-90", Name: "LIMPEZA LTDA", Acts: 3}
	src := &fakeSuggestionSource{companies: map[string]domain.Suggestion{"12345678000190": company}}

	got, err := NewSuggest(src).Execute(context.Background(), "12.345.678/0001-90")

	if err != nil || !reflect.DeepEqual(got, []domain.Suggestion{company}) {
		t.Fatalf("CNPJ citado vira a sugestão da empresa: %+v %v", got, err)
	}
}

func TestSuggestKeepsOnlyNumbersCitedInActs(t *testing.T) {
	src := &fakeSuggestionSource{entities: map[string]int{"contrato:7/2015": 4}}

	got, err := NewSuggest(src).Execute(context.Background(), "007/2015")

	want := []domain.Suggestion{{Kind: domain.EntityContrato, Key: "7/2015", Label: "007/2015", Acts: 4}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("o processo 0072015 não tem ato e fica de fora: %+v %v", got, err)
	}
}

func TestSuggestSearchesCompanyNames(t *testing.T) {
	match := domain.Suggestion{Kind: domain.EntityCNPJ, Key: "12345678000190", Label: "12.345.678/0001-90", Name: "LIMPEZA LTDA", Acts: 3}
	src := &fakeSuggestionSource{byName: []domain.Suggestion{match}}

	got, err := NewSuggest(src).Execute(context.Background(), "  limpeza ")

	if err != nil || src.nameQuery != "limpeza" || !reflect.DeepEqual(got, []domain.Suggestion{match}) {
		t.Fatalf("texto busca pelo nome da empresa: %q %+v %v", src.nameQuery, got, err)
	}
}

func TestSuggestReturnsNothingForShortQueries(t *testing.T) {
	got, err := NewSuggest(&fakeSuggestionSource{}).Execute(context.Background(), "ab")

	if err != nil || len(got) != 0 {
		t.Fatalf("consulta curta não consulta o banco: %+v %v", got, err)
	}
}

func TestSuggestPropagatesErrors(t *testing.T) {
	boom := errors.New("boom")

	_, err := NewSuggest(&fakeSuggestionSource{err: boom}).Execute(context.Background(), "limpeza")

	if !errors.Is(err, boom) {
		t.Fatalf("erro do banco sobe: %v", err)
	}
}
