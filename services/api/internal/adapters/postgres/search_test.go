package postgres

import "testing"

func TestLikePatternEscapesWildcards(t *testing.T) {
	if got := likePattern(`50%_a\b`); got != `%50\%\_a\\b%` {
		t.Errorf("padrão inesperado: %q", got)
	}
}

func TestHighlightFallback(t *testing.T) {
	cases := []struct{ snippet, query, want string }{
		{"Nomeia JOSÉ DA SILVA para o cargo", "jose da silva", "Nomeia ⟦JOSÉ DA SILVA⟧ para o cargo"},
		{"CNPJ: 51.903.675/0001-81.", "903.675/0001", "CNPJ: 51.⟦903.675/0001⟧-81."},
		{"já ⟦marcado⟧ pelo banco", "marcado", "já ⟦marcado⟧ pelo banco"},
		{"sem ocorrência", "xyz", "sem ocorrência"},
		{"Ação social", "acao", "⟦Ação⟧ social"},
	}
	for _, c := range cases {
		if got := highlightFallback(c.snippet, c.query); got != c.want {
			t.Errorf("highlightFallback(%q, %q) = %q, esperava %q", c.snippet, c.query, got, c.want)
		}
	}
}
