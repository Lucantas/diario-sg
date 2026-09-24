package domain

import "testing"

func TestEntityLabel(t *testing.T) {
	cases := []struct {
		kind  EntityKind
		value string
		want  string
	}{
		{EntityProcesso, "PROCESSO ADMINISTRATIVO N.º 7148/2022", "7148/2022"},
		{EntityProcesso, "Processo n.º 14.672/2021", "14.672/2021"},
		{EntityProcesso, "Processo\n\n03.06524/2022-4", "03.06524/2022-4"},
		{EntityProcesso, "processo SEI Nº\n25.00383/2026-5", "25.00383/2026-5"},
		{EntityContrato, "Contrato PMSG Nº.\n001/2016", "001/2016"},
		{EntityContrato, "CONTRATO Nº 30/FMS/2011", "30/FMS/2011"},
		{EntityContrato, "Contrato nº 007/2024/SEMAD.", "007/2024/SEMAD"},
		{EntityCNPJ, "12345678000190", "12.345.678/0001-90"},
	}
	for _, c := range cases {
		if got := EntityLabel(c.kind, c.value); got != c.want {
			t.Errorf("%s %q: esperava %q, veio %q", c.kind, c.value, c.want, got)
		}
	}
}

func TestEntitySlugRoundTripsThroughParse(t *testing.T) {
	for kind, label := range map[EntityKind]string{EntityProcesso: "14.672/2021", EntityContrato: "30/FMS/2011"} {
		want, _ := ParseEntityInput(kind, label)
		got, err := ParseEntityInput(kind, EntitySlug(label))
		if err != nil || got != want {
			t.Errorf("%s %q: slug %q deu %q (%v), esperava %q", kind, label, EntitySlug(label), got, err, want)
		}
	}
}
