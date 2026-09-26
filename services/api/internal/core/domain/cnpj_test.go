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

func TestHasValidCheckDigits(t *testing.T) {
	for cnpj, want := range map[string]bool{"28636579000100": true, "14180324000163": true, "28579636000100": false, "1234": false} {
		if got := HasValidCheckDigits(cnpj); got != want {
			t.Errorf("%s: esperava %v", cnpj, want)
		}
	}
}

func TestMunicipalOrgCNPJsAreValidAndStartWithTheMunicipality(t *testing.T) {
	orgs := MunicipalOrgCNPJs()

	if orgs[0] != "28636579000100" {
		t.Fatalf("primeiro órgão: %s", orgs[0])
	}
	for _, o := range orgs {
		if !HasValidCheckDigits(o) {
			t.Errorf("CNPJ inválido na lista: %s", o)
		}
	}
}
