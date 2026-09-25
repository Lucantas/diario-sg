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
		{"com base no art. 24, inciso I da Lei 9.394/96 (LDB)", nil},
		{"nos termos do art. 24, I, do Código de Trânsito Brasileiro", nil},
		{"Lei Federal nº 8.666/93, art. 24, inciso IV", []LegalBasis{Art24IV}},
		{"fica vedada a recontratação de empresa já contratada com base no inciso VIII do art. 75 da Lei 14.133", nil},
		{"na hipótese de contratação direta fundamentada no art. 75, VIII, da Lei nº 14.133", nil},
		{"art. 75, inciso III da Lei 14.133", nil},
		{"art. 24 da Lei 8.666", nil},
		{"Art. 24. Inciso XXII", nil},
		{"FUNDAMENTO LEGAL: ARTIGO 24, INCISOS I E II DA LEI FEDERAL N.º 8.666/93.", []LegalBasis{Art24I, Art24II}},
		{"art. 24, incisos II e IV, da Lei 8.666/93", []LegalBasis{Art24II, Art24IV}},
		{"art. 24, I, II e IV da Lei 8.666/93", []LegalBasis{Art24I, Art24II, Art24IV}},
		{"prevista nos incisos I e II do art. 75 da Lei Federal nº 14.133/2021", []LegalBasis{Art75I, Art75II}},
		{"incisos I e II, do artigo 24 da Lei 8.666", []LegalBasis{Art24I, Art24II}},
		{"de acordo com o art.24, incisos II e III, da Lei nº. 9.503, de 23 de setembro de 1997", nil},
		{"art. 24, incisos II e XIII, da Lei 8.666/93", []LegalBasis{Art24II}},
		{"art. 24, II, e art. 26 da Lei 8.666/93", []LegalBasis{Art24II}},
	}
	for _, c := range cases {
		if got := LegalBasisOf(c.body); !slices.Equal(got, c.want) {
			t.Errorf("%q: veio %v, esperava %v", c.body, got, c.want)
		}
	}
}

func TestCitesWorksDispensa(t *testing.T) {
	if !CitesWorksDispensa("art. 24, inciso I, da Lei 8.666/93") || !CitesWorksDispensa("inciso I do artigo 75 da Lei 14.133") ||
		CitesWorksDispensa("art. 24, inciso II da Lei 8.666") || CitesWorksDispensa("art. 24, inciso IV da Lei 8.666") {
		t.Error("CitesWorksDispensa")
	}
}
