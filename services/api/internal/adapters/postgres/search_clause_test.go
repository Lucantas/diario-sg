package postgres

import (
	"strings"
	"testing"
)

func TestMatchForBindsBothParameters(t *testing.T) {
	got := matchFor("$2", "$3")
	if strings.Contains(got, "$Q") || strings.Contains(got, "$LIKE") || !strings.Contains(got, "length($2)") || !strings.Contains(got, "unaccent_immutable($3)") {
		t.Errorf("cláusula mal instanciada: %s", got)
	}
}

func TestLikePatternIgnoresQuotesAndEscapesWildcards(t *testing.T) {
	cases := map[string]string{
		`"jose da silva"`: "%jose da silva%",
		` "  jose "  `:    "%jose%",
		`100%_a\b`:        `%100\%\_a\\b%`,
	}
	for in, want := range cases {
		if got := likePattern(in); got != want {
			t.Errorf("likePattern(%q) = %q, esperava %q", in, got, want)
		}
	}
}

func TestExactPhraseForBindsParameter(t *testing.T) {
	got := exactPhraseFor("$7")
	if strings.Contains(got, "$LIKE") || !strings.Contains(got, "unaccent_immutable($7)") || !strings.Contains(got, "AS exact") {
		t.Errorf("join mal instanciado: %s", got)
	}
}
