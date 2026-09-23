package domain

import "testing"

func TestNormalizeCNPJ(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"12.345.678/0001-90", "12345678000190", true},
		{"12345678000190", "12345678000190", true},
		{"12.345.678/0001- 90", "12345678000190", true},
		{"12.345.678/0001-9", "", false},
		{"12.345.678/0001-901", "", false},
		{"abc", "", false},
	}
	for _, c := range cases {
		got, ok := NormalizeCNPJ(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("NormalizeCNPJ(%q) = %q,%v; esperava %q,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestNormalizeCNPJKeepsWrongCheckDigitsBecauseTheGazettePublishesTypos(t *testing.T) {
	got, ok := NormalizeCNPJ("12.345.678/0001-00")

	if !ok || got != "12345678000100" {
		t.Errorf("NormalizeCNPJ = %q,%v; esperava 12345678000100,true", got, ok)
	}
}
