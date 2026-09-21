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
