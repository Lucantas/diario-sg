package domain

import (
	"errors"
	"testing"
)

func TestTranslateOperators(t *testing.T) {
	cases := map[string]string{
		"limpeza OU coleta":          "limpeza or coleta",
		"limpeza ou coleta":          "limpeza ou coleta",
		"OUTORGA":                    "OUTORGA",
		`"carne OU frango" OU peixe`: `"carne OU frango" or peixe`,
		"a OU b OU c":                "a or b or c",
		"OU":                         "or",
		"":                           "",
	}
	for in, want := range cases {
		if got := TranslateOperators(in); got != want {
			t.Errorf("TranslateOperators(%q) = %q, esperava %q", in, got, want)
		}
	}
}

func TestFilterValueRange(t *testing.T) {
	for _, f := range []ActFilter{{MinCents: -1}, {MaxCents: -1}, {MinCents: 500, MaxCents: 100}} {
		if err := f.Normalize(); !errors.Is(err, ErrInvalidFilter) {
			t.Errorf("%+v: esperava ErrInvalidFilter, veio %v", f, err)
		}
	}
	f := ActFilter{MinCents: 100, MaxCents: 100, Query: "a OU b"}
	if err := f.Normalize(); err != nil || f.Query != "a or b" {
		t.Errorf("veio %+v %v", f, err)
	}
	f = ActFilter{MinCents: 500}
	if err := f.Normalize(); err != nil {
		t.Errorf("só mínimo é válido: %v", err)
	}
}
