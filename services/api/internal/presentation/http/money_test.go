package http

import "testing"

func TestParseReais(t *testing.T) {
	ok := map[string]int64{"": 0, "1500": 150000, "1500.5": 150050, "1500.50": 150050, "0.01": 1}
	for in, want := range ok {
		if got, err := parseReais(in); err != nil || got != want {
			t.Errorf("parseReais(%q) = %d, %v; esperava %d", in, got, err, want)
		}
	}
	for _, in := range []string{"1.500,50", "1,5", "-3", "abc", "1.505", "1e3", "99999999999999"} {
		if _, err := parseReais(in); err == nil {
			t.Errorf("parseReais(%q) deveria falhar", in)
		}
	}
}
