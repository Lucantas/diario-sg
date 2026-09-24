package domain

import "testing"

func TestPublicBodyNamesTheMunicipalityAndItsBranches(t *testing.T) {
	cases := map[string]string{
		"28.636.579/0001-00": "Município de São Gonçalo",
		"28636579000950":     "Município de São Gonçalo",
		"28.579.636/0001-00": "Município de São Gonçalo",
		"32.538.167/0001-05": "SG-PREVI",
		"29.846.003/0001-22": "Câmara Municipal de São Gonçalo",
		"10.746.140/0001-67": "",
		"não é cnpj":         "",
	}
	for cnpj, want := range cases {
		got, ok := PublicBody(cnpj)
		if got != want || ok != (want != "") {
			t.Errorf("%s: esperava %q, veio %q (%v)", cnpj, want, got, ok)
		}
	}
}
