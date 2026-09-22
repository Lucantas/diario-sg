package parser

import "testing"

func TestEditionNumber(t *testing.T) {
	cases := map[string]string{
		"PODER EXECUTIVO (D.O.E) | ANO VII\n\n18 DE SETEMBRO DE 2026 | EDIÇÃO N°1.771\n":                                                   "1771",
		"DIÁRIO OFICIAL ELETRÔNICO DO MUNICÍPIO DE SÃO GONÇALO D.O.E. | PODER EXECUTIVO | ANO V | N.º 1.062 EM 15 DE MARÇO DE 2024":        "1062",
		"10 DE MARÇO DE 2025 | EDIÇÃO N°1. 362\n":                                                                                          "1362",
		"Diário Oficial Eletrônico do Município de São Gonçalo - D.O.E. - | Poder Executivo | Ano I | N.º 190 | em 08 de outubro de 2020.": "190",
		"Diário Oficial Eletrônico do Município de São Gonçalo - D.O.E. - | Poder Executivo | Ano II | N.º 301 | em 10 de março de 2021.":  "301",
		"LEI Nº 1164/2020\nDecreto nº 63/2020, de 16 de março de 2020":                                                                     "",
		"texto sem número de edição": "",
	}
	for text, want := range cases {
		if got := New().EditionNumber(text); got != want {
			t.Errorf("EditionNumber(%q) = %q, esperava %q", text, got, want)
		}
	}
}
