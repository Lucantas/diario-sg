package domain

import (
	"reflect"
	"testing"
)

func TestOrganVariantsPointToKnownPrincipals(t *testing.T) {
	for variant, principal := range organVariantOf {
		if !IsKnownOrgan(variant) || !IsKnownOrgan(principal) {
			t.Errorf("%s → %s: as duas siglas precisam estar no catálogo", variant, principal)
		}
		if _, chained := organVariantOf[principal]; chained {
			t.Errorf("%s → %s: a principal não pode ser variante de outra", variant, principal)
		}
	}
}

func TestNormalizeOrganUsesThePrincipalAcronym(t *testing.T) {
	for in, want := range map[string]string{"semsad": "SEMSADC", "SAMSADC": "SEMSADC", "SEMSADC": "SEMSADC", "FMSSG": "FMS", "SEMED": "SEMED"} {
		if got, ok := NormalizeOrgan(in); got != want || !ok {
			t.Errorf("NormalizeOrgan(%q) = %q, %v; esperava %q", in, got, ok, want)
		}
	}
}

func TestOrganAcronymsIncludeTheVariants(t *testing.T) {
	cases := map[string][]string{
		"SEMSADC":   {"SAMSADC", "SEMSAD", "SEMSADC"},
		"SEMGOVCOM": {"SEMGOVCOM", "SEMGOVCOMS", "SEMGOVCON"},
		"SEMED":     {"SEMED"},
	}
	for principal, want := range cases {
		if got := OrganAcronyms(principal); !reflect.DeepEqual(got, want) {
			t.Errorf("OrganAcronyms(%q) = %v, esperava %v", principal, got, want)
		}
	}
}
