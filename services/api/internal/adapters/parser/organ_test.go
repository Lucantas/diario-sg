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

func organs(text string) string {
	var got []string
	for _, a := range New().Parse("ATOS DO PREFEITO\n" + text) {
		got = append(got, a.Title+"="+a.Organ)
	}
	return strings.Join(got, "|")
}

func TestParse_AbbreviatedPortariaBelongsToTheCabinetNotThePreviousOrgan(t *testing.T) {
	text := "SUBCOMP\nAVISO DE LICITAÇÃO\nPregão Presencial nº 1/2016.\n" +
		"Designa:\na contar de 01 de agosto de 2016, FULANO, para responder pela função.\n" +
		"Port. nº 1360/2016\n" +
		"EXTRATO DO CONTRATO Nº 3/2016\nObjeto: obras."

	want := "AVISO DE LICITAÇÃO=SUBCOMP|Port. nº 1360/2016=|EXTRATO DO CONTRATO Nº 3/2016="
	if got := organs(text); got != want {
		t.Errorf("esperava\n%s\nveio\n%s", want, got)
	}
}

func TestParse_ContinuationOfThePersonnelAnnexClearsTheOrgan(t *testing.T) {
	text := "FMS\nEXTRATO DO CONTRATO Nº 3/2025\nObjeto: medicamentos.\n" +
		"Continuação do D.O.E. em 17/04/2025\n" +
		"PORTARIA Nº 20/2025\nDispõe sobre o ponto facultativo."

	want := "EXTRATO DO CONTRATO Nº 3/2025=FMS|PORTARIA Nº 20/2025="
	if got := organs(text); got != want {
		t.Errorf("esperava\n%s\nveio\n%s", want, got)
	}
}

func TestParse_WordShapedLikeAnAcronymIsNotAnOrgan(t *testing.T) {
	text := "SEMAD\nPORTARIA Nº 10/2016\nConcede licença.\n" +
		"MENSAGEM Nº 025/2015 DE AUTORIA DO PODER\nEXECUTIVO\n" +
		"DECRETO Nº 124/2016\nDisciplina a extinção de cargo.\n" +
		"TOTAL\nEDITAL DE CONVOCAÇÃO\nConvoca os aprovados."

	want := "PORTARIA Nº 10/2016=SEMAD|DECRETO Nº 124/2016=|EDITAL DE CONVOCAÇÃO="
	if got := organs(text); got != want {
		t.Errorf("esperava\n%s\nveio\n%s", want, got)
	}
}

func TestParse_ResolutionWithQuotedSeriesLetterOpensTheOrganSection(t *testing.T) {
	text := "SUBCOMP\nAVISO DE LICITAÇÃO\nPregão.\n" +
		"CMS\nRESOLUÇÃO “P” nº 015/CMS-SG/16\nAprova o plano.\n" +
		"ATA DA REUNIÃO ORDINÁRIA DO CONSELHO MUNICIPAL\nAos dez dias."

	want := "AVISO DE LICITAÇÃO=SUBCOMP|RESOLUÇÃO “P” nº 015/CMS-SG/16=CMS|ATA DA REUNIÃO ORDINÁRIA DO CONSELHO MUNICIPAL=CMS"
	if got := organs(text); got != want {
		t.Errorf("esperava\n%s\nveio\n%s", want, got)
	}
}
