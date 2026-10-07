package domain

import "testing"

func TestParseLookup(t *testing.T) {
	cases := []struct {
		q    string
		want Lookup
	}{
		{"12.345.678/0001-90", Lookup{CNPJ: "12345678000190"}},
		{"12345678000190", Lookup{CNPJ: "12345678000190"}},
		{"4.321/2024", Lookup{Processo: "43212024", ProcessoLabel: "4.321/2024"}},
		{"7/2015", Lookup{Processo: "72015", ProcessoLabel: "7/2015", Contrato: "7/2015", ContratoLabel: "7/2015"}},
		{"007/2015", Lookup{Processo: "0072015", ProcessoLabel: "007/2015", Contrato: "7/2015", ContratoLabel: "007/2015"}},
		{"12345", Lookup{Processo: "12345", ProcessoLabel: "12345"}},
		{"30/fms/2011", Lookup{Contrato: "30/FMS/2011", ContratoLabel: "30/FMS/2011"}},
		{"processo 06.10981/2025-7", Lookup{Processo: "061098120257", ProcessoLabel: "06.10981/2025-7"}},
		{"Processo nº 4.321/2024", Lookup{Processo: "43212024", ProcessoLabel: "4.321/2024"}},
		{"processo 7/2015", Lookup{Processo: "72015", ProcessoLabel: "7/2015"}},
		{"contrato nº 30/FMS/2011", Lookup{Contrato: "30/FMS/2011", ContratoLabel: "30/FMS/2011"}},
		{"CONTRATO N.º 7/2015", Lookup{Contrato: "7/2015", ContratoLabel: "7/2015"}},
		{"contrato 4.321/2024", Lookup{}},
		{"processo de licitação", Lookup{}},
		{"limpeza urbana", Lookup{Name: "limpeza urbana"}},
		{"  Ação Social  ", Lookup{Name: "Ação Social"}},
		{"ab", Lookup{}},
		{"12", Lookup{}},
		{"1234", Lookup{}},
		{"", Lookup{}},
	}
	for _, c := range cases {
		if got := ParseLookup(c.q); got != c.want {
			t.Errorf("ParseLookup(%q) = %+v, esperava %+v", c.q, got, c.want)
		}
	}
}

func TestParseLookupIgnoresTooLongQueries(t *testing.T) {
	long := make([]byte, maxLookupLength+1)
	for i := range long {
		long[i] = 'a'
	}
	if got := ParseLookup(string(long)); got != (Lookup{}) {
		t.Errorf("consulta longa demais não vira sugestão: %+v", got)
	}
}

func TestLikePatternEscapesWildcards(t *testing.T) {
	if got := ContainsPattern(`50%_off\x`); got != `%50\%\_off\\x%` {
		t.Errorf("ContainsPattern escapa curinga do LIKE: %q", got)
	}
}
