package parser

import "testing"

func TestEditionNumber(t *testing.T) {
	cases := map[string]string{
		"PODER EXECUTIVO (D.O.E) | ANO VII\n\n18 DE SETEMBRO DE 2026 | EDIÇÃO N°1.771\n":                                            "1771",
		"DIÁRIO OFICIAL ELETRÔNICO DO MUNICÍPIO DE SÃO GONÇALO D.O.E. | PODER EXECUTIVO | ANO V | N.º 1.062 EM 15 DE MARÇO DE 2024": "1062",
		"texto sem número de edição": "",
	}
	for text, want := range cases {
		if got := New().EditionNumber(text); got != want {
			t.Errorf("EditionNumber(%q) = %q, esperava %q", text, got, want)
		}
	}
}
