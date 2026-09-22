package parser

import (
	"strings"
	"testing"
)

func TestParse_OrganComesFromTheSectionAcronym(t *testing.T) {
	text := "ATOS DO PREFEITO\n" +
		"DECRETO Nº 1/2026\nDispõe sobre o horário.\n" +
		"SEMAD\nPORTARIA Nº 10/2026\nNomeia servidor para a função.\n" +
		"PORTARIA Nº 11/2026\nExonera servidor da função.\n" +
		"FMS\nEXTRATO DO CONTRATO Nº 3/2026\nObjeto: medicamentos.\n" +
		"ANEXO\nDECRETO Nº 2/2026\nDispõe sobre a feira."

	acts := New().Parse(text)

	var got []string
	for _, a := range acts {
		got = append(got, a.Title+"="+a.Organ)
	}
	want := []string{
		"DECRETO Nº 1/2026=",
		"PORTARIA Nº 10/2026=SEMAD",
		"PORTARIA Nº 11/2026=SEMAD",
		"EXTRATO DO CONTRATO Nº 3/2026=FMS",
		"DECRETO Nº 2/2026=FMS",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("esperava\n%v\nveio\n%v", want, got)
	}
}
