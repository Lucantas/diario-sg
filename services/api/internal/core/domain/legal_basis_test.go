package domain

import (
	"slices"
	"testing"
)

func TestLegalBasisOf(t *testing.T) {
	cases := []struct {
		body string
		want []LegalBasis
	}{
		{"RATIFICO a situação de dispensa de licitação com fundamento no art. 24, inciso IV, da Lei Federal nº 8.666/93", []LegalBasis{Art24IV}},
		{"Espécie: Emergencial: Base Legal: art. 24, inc. IV, da Lei nº 8666/93.", []LegalBasis{Art24IV}},
		{"dispensa com fundamento no inciso VIII do art. 75 da Lei 14.133/2021", []LegalBasis{Art75VIII}},
		{"nos termos do Art. 75, VIII, da Lei 14.133", []LegalBasis{Art75VIII}},
		{"art. 24, inciso I, da Lei 8.666/93 e art. 24, inciso II", []LegalBasis{Art24I, Art24II}},
		{"conforme artigo 75, inciso I da Lei de Licitações n.º 14.133", []LegalBasis{Art75I}},
		{"art. 24, inciso XIII, da Lei Federal nº 8.666/93", nil},
		{"art. 75, inciso III", nil},
		{"art. 24 da Lei 8.666", nil},
		{"Art. 24. Inciso XXII", nil},
	}
	for _, c := range cases {
		if got := LegalBasisOf(c.body); !slices.Equal(got, c.want) {
			t.Errorf("%q: veio %v, esperava %v", c.body, got, c.want)
		}
	}
}

func TestCitesWorksDispensa(t *testing.T) {
	if !CitesWorksDispensa("art. 24, inciso I, da Lei 8.666/93") || !CitesWorksDispensa("inciso I do artigo 75") ||
		CitesWorksDispensa("art. 24, inciso II") || CitesWorksDispensa("art. 24, inciso IV") {
		t.Error("CitesWorksDispensa")
	}
}
