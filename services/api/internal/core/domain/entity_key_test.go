package domain

import (
	"errors"
	"testing"
)

func TestEntityKeyAndCertainty(t *testing.T) {
	cases := []struct {
		kind       EntityKind
		normalized string
		key        string
		certainty  Certainty
	}{
		{EntityCNPJ, "28636579000100", "28636579000100", CertaintyExact},
		{EntityProcesso, "061098120257", "061098120257", CertaintyStrong},
		{EntityContrato, "001/2017", "1/2017", CertaintyWeak},
		{EntityContrato, "30/FMS/2011", "30/FMS/2011", CertaintyStrong},
		{EntityContrato, "0/2020", "0/2020", CertaintyWeak},
		{EntityContrato, "007/2024/SEMAD", "7/2024/SEMAD", CertaintyStrong},
	}
	for _, c := range cases {
		key := EntityKey(c.kind, c.normalized)
		if key != c.key {
			t.Errorf("EntityKey(%s, %q) = %q, esperava %q", c.kind, c.normalized, key, c.key)
		}
		if got := LinkCertainty(c.kind, key); got != c.certainty {
			t.Errorf("LinkCertainty(%s, %q) = %q, esperava %q", c.kind, key, got, c.certainty)
		}
	}
}

func TestParseEntityInput(t *testing.T) {
	ok := []struct {
		kind EntityKind
		in   string
		want string
	}{
		{EntityCNPJ, "28.636.579/0001-00", "28636579000100"},
		{EntityProcesso, "06.10981/2025-7", "061098120257"},
		{EntityProcesso, " 8421/2023 ", "84212023"},
		{EntityContrato, "001/2017", "1/2017"},
		{EntityContrato, "30 / fms / 2011", "30/FMS/2011"},
	}
	for _, c := range ok {
		if got, err := ParseEntityInput(c.kind, c.in); err != nil || got != c.want {
			t.Errorf("ParseEntityInput(%s, %q) = %q, %v; esperava %q", c.kind, c.in, got, err, c.want)
		}
	}
	bad := []struct {
		kind EntityKind
		in   string
		want error
	}{
		{EntityCNPJ, "123", ErrInvalidCNPJ},
		{EntityProcesso, "12/3", ErrInvalidInput},
		{EntityContrato, "contrato", ErrInvalidInput},
		{EntityValor, "100", ErrInvalidInput},
	}
	for _, c := range bad {
		if _, err := ParseEntityInput(c.kind, c.in); !errors.Is(err, c.want) {
			t.Errorf("ParseEntityInput(%s, %q): esperava %v, veio %v", c.kind, c.in, c.want, err)
		}
	}
}

func TestLinkedKindsExcludeValues(t *testing.T) {
	for kind, want := range map[EntityKind]bool{EntityCNPJ: true, EntityProcesso: true, EntityContrato: true, EntityValor: false} {
		if IsLinkedKind(kind) != want {
			t.Errorf("IsLinkedKind(%s) deveria ser %v", kind, want)
		}
	}
}
