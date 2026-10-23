package domain

import "testing"

func TestGazetteFingerprintJoinsFieldsRowsAndTables(t *testing.T) {
	acts := [][]string{{"a1", "g1", "x"}, {"a2", "g1", "y"}}
	entities := [][]string{{"a1", "cnpj", "1"}}

	got := FingerprintOf(acts, entities)

	if got != "42ed7aa83b4066d91bc69bb37669331d" {
		t.Fatalf("impressão digital inesperada: %s", got)
	}
}

func TestGazetteFingerprintOfGazetteWithoutActs(t *testing.T) {
	if got := FingerprintOf(nil, nil); got != "eb259edbaa608eb2208046619b484668" {
		t.Fatalf("impressão digital inesperada: %s", got)
	}
}

func TestGazetteFingerprintChangesWhenAnyFieldChanges(t *testing.T) {
	base := FingerprintOf([][]string{{"a1", "corpo"}}, nil)

	if FingerprintOf([][]string{{"a1", "corpo."}}, nil) == base {
		t.Fatal("mudança no corpo deve mudar a impressão digital")
	}
	if FingerprintOf([][]string{{"a1", "corpo"}}, [][]string{{"a1"}}) == base {
		t.Fatal("entidade nova deve mudar a impressão digital")
	}
}
