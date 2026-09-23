package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeOrgan(t *testing.T) {
	cases := map[string]struct {
		want string
		ok   bool
	}{
		"SEMED": {"SEMED", true}, " semed ": {"SEMED", true}, "": {"", true},
		"TOTAL": {"", false}, "SEM ED": {"", false},
	}
	for in, c := range cases {
		got, ok := NormalizeOrgan(in)
		if got != c.want || ok != c.ok {
			t.Errorf("NormalizeOrgan(%q) = %q, %v; esperava %q, %v", in, got, ok, c.want, c.ok)
		}
	}
}

func TestOrganCatalogIsWellFormed(t *testing.T) {
	if len(organNames) != 125 {
		t.Fatalf("esperava as 125 siglas conhecidas, veio %d", len(organNames))
	}
	for acronym, name := range organNames {
		if acronym != strings.ToUpper(acronym) || strings.ContainsAny(acronym, " \t") {
			t.Errorf("sigla fora do formato: %q", acronym)
		}
		if name != strings.TrimSpace(name) {
			t.Errorf("nome de %s com espaço nas pontas: %q", acronym, name)
		}
	}
}

func TestOrganNameSpellsOutTheAcronym(t *testing.T) {
	cases := map[string]string{
		"SEMED": "Secretaria Municipal de Educação",
		"SEMAS": "Secretaria Municipal de Assistência Social",
		"SMC":   "",
		"TOTAL": "",
	}
	for acronym, want := range cases {
		if got := OrganName(acronym); got != want {
			t.Errorf("OrganName(%q) = %q; esperava %q", acronym, got, want)
		}
	}
	named := 0
	for _, name := range organNames {
		if name != "" {
			named++
		}
	}
	if named != 119 {
		t.Fatalf("esperava 119 siglas com nome (ver docs/orgaos.md), veio %d", named)
	}
}

func TestFilterNormalizesAndValidatesOrgan(t *testing.T) {
	f := ActFilter{Organ: "semed"}
	if err := f.Normalize(); err != nil || f.Organ != "SEMED" {
		t.Fatalf("esperava SEMED sem erro, veio %q %v", f.Organ, err)
	}
	f = ActFilter{Organ: "TOTAL"}
	if err := f.Normalize(); !errors.Is(err, ErrInvalidFilter) {
		t.Fatalf("esperava ErrInvalidFilter, veio %v", err)
	}
}
