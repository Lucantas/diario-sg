package domain

import (
	"reflect"
	"testing"
)

func TestActWarnings(t *testing.T) {
	cases := []struct {
		name  string
		facts WarningFacts
		want  []string
	}{
		{"portaria sem número", WarningFacts{Title: "Nomeia:", PageStart: 1, PageEnd: 1}, []string{WarningNoNumber}},
		{"verbo com complemento", WarningFacts{Title: "Exonera a pedido:", PageStart: 1, PageEnd: 1}, []string{WarningNoNumber}},
		{"portaria com número", WarningFacts{Title: "Port. nº 12/2026", PageStart: 1, PageEnd: 1}, []string{}},
		{"só título", WarningFacts{Title: "PORTARIA Nº 1/2026", TitleOnly: true, PageStart: 1, PageEnd: 1}, []string{WarningTitleOnly}},
		{"dez páginas", WarningFacts{Title: "EDITAL", PageStart: 3, PageEnd: 12}, []string{WarningManyPages}},
		{"nove páginas", WarningFacts{Title: "EDITAL", PageStart: 3, PageEnd: 11}, []string{}},
		{"página desconhecida", WarningFacts{Title: "EDITAL"}, []string{}},
		{"uma assinatura", WarningFacts{Title: "EDITAL", Signatures: 1}, []string{}},
		{"duas assinaturas", WarningFacts{Title: "EDITAL", Signatures: 2}, []string{WarningManyActs}},
		{"tudo junto", WarningFacts{Title: "Designa:", TitleOnly: true, Signatures: 3, PageStart: 1, PageEnd: 40},
			[]string{WarningNoNumber, WarningTitleOnly, WarningManyPages, WarningManyActs}},
	}
	for _, c := range cases {
		got := ActWarnings(c.facts)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: esperava %v, veio %#v", c.name, c.want, got)
		}
	}
}

func TestWarningFactsOfActReadsTheBody(t *testing.T) {
	a := Act{Title: "PORTARIA Nº 134/2014", PageStart: 2, PageEnd: 2, Body: `PORTARIA Nº 134/2014
Exclui servidores da comissão.
São Gonçalo, 18 de julho de 2014.
ROSELI CONSTANTINO
Portaria nº 159/SUPES/SEMAD/2014
Averba tempo de serviço.
São Gonçalo, 21 de Julho de 2014.`}

	got := WarningFactsOf(a)

	if got.Signatures != 2 || got.TitleOnly || got.Title != a.Title || got.PageStart != 2 {
		t.Fatalf("fatos errados: %+v", got)
	}
}

func TestCountSignatures(t *testing.T) {
	cases := map[string]int{
		"sem data":                                     0,
		"São Gonçalo, 1º de março de 2026.":            1,
		"São Gonçalo,03 de julho de 2024":              1,
		"SÃO GONÇALO, 3 DE JULHO DE 2024. São Gonçalo": 1,
		"Lei de 01 de dezembro de 2025":                0,
	}
	for body, want := range cases {
		if got := CountSignatures(body); got != want {
			t.Errorf("%q: esperava %d, veio %d", body, want, got)
		}
	}
}

func TestIsTitleOnly(t *testing.T) {
	if !IsTitleOnly("A", " A \n") {
		t.Error("corpo igual ao título, com espaços, é só título")
	}
	if IsTitleOnly("A", "A\nB") {
		t.Error("corpo com mais texto não é só título")
	}
}
