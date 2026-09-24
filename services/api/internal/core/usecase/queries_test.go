package usecase

import (
	"context"
	"reflect"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type organCounts struct {
	ports.ActRepository
	counts map[string]int
}

func (o organCounts) CountByOrgan(context.Context) (map[string]int, error) { return o.counts, nil }

func TestListOrgansSortsByActsThenAcronymAndNamesThem(t *testing.T) {
	uc := NewListOrgans(organCounts{counts: map[string]int{"SEMED": 3, "FMS": 5, "SEMAD": 3}})

	got, err := uc.Execute(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	want := []domain.OrganListing{
		{Organ: domain.Organ{Acronym: "FMS", Name: domain.OrganName("FMS")}, Acts: 5},
		{Organ: domain.Organ{Acronym: "SEMAD", Name: domain.OrganName("SEMAD")}, Acts: 3},
		{Organ: domain.Organ{Acronym: "SEMED", Name: domain.OrganName("SEMED")}, Acts: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("esperava %+v, veio %+v", want, got)
	}
}

func TestListOrgansAddsVariantsToThePrincipal(t *testing.T) {
	uc := NewListOrgans(organCounts{counts: map[string]int{"FMS": 5, "FMSSG": 2, "SEMSAD": 1}})

	got, err := uc.Execute(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	want := []domain.OrganListing{
		{Organ: domain.Organ{Acronym: "FMS", Name: domain.OrganName("FMS")}, Acts: 7},
		{Organ: domain.Organ{Acronym: "SEMSADC", Name: domain.OrganName("SEMSADC")}, Acts: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("esperava %+v, veio %+v", want, got)
	}
}
