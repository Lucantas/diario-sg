package domain

import (
	"errors"
	"testing"
	"time"
)

var alertNow = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

func TestSubscriptionWithFiltersOnly(t *testing.T) {
	s, err := NewSubscription("a@b.com", "", AlertFilter{Type: ActLicencaAmbiental, Organ: "semmatran", Theme: "meio_ambiente"}, alertNow)

	if err != nil || s.Filter.Organ != "SEMMATRAN" || s.Query != "" {
		t.Fatalf("veio %+v %v", s, err)
	}
	if got := s.Subject(); got != "licença ambiental · SEMMATRAN · meio ambiente" {
		t.Fatalf("assunto: %q", got)
	}
}

func TestSubscriptionWithTermAndFilter(t *testing.T) {
	s, err := NewSubscription("a@b.com", "  vista   alegre ", AlertFilter{Source: SourceDiarioPrefeitura}, alertNow)

	if err != nil || s.Subject() != "“vista alegre” · Diário da Prefeitura" {
		t.Fatalf("veio %q %v", s.Subject(), err)
	}
}

func TestSubscriptionNeedsTermOrFilterAndValidFilters(t *testing.T) {
	cases := []struct {
		query  string
		filter AlertFilter
		want   error
	}{
		{"", AlertFilter{}, ErrInvalidQuery},
		{"ab", AlertFilter{}, ErrInvalidQuery},
		{"", AlertFilter{Type: "multa"}, ErrInvalidFilter},
		{"", AlertFilter{Organ: "TOTAL"}, ErrInvalidFilter},
		{"", AlertFilter{Theme: "saude"}, ErrInvalidFilter},
		{"", AlertFilter{Source: "diario_estado"}, ErrInvalidFilter},
	}
	for _, c := range cases {
		if _, err := NewSubscription("a@b.com", c.query, c.filter, alertNow); !errors.Is(err, c.want) {
			t.Errorf("%q %+v: esperava %v, veio %v", c.query, c.filter, c.want, err)
		}
	}
}

func TestAlertFilterBuildsTheSearchOfOneEdition(t *testing.T) {
	f := AlertFilter{Type: ActLicencaAmbiental, Theme: "meio_ambiente"}.ActFilter("licença", "g-1")

	if f.GazetteID != "g-1" || f.Query != "licença" || f.Type != ActLicencaAmbiental || f.Theme != "meio_ambiente" || f.Limit != alertHitsPerEdition {
		t.Fatalf("filtro: %+v", f)
	}
}
