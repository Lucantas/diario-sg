package domain

import (
	"reflect"
	"testing"
)

func TestActWarnings(t *testing.T) {
	cases := []struct {
		name      string
		title     string
		titleOnly bool
		start     int
		end       int
		want      []string
	}{
		{"portaria sem número", "Nomeia:", false, 1, 1, []string{WarningNoNumber}},
		{"verbo com complemento", "Exonera a pedido:", false, 1, 1, []string{WarningNoNumber}},
		{"portaria com número", "Port. nº 12/2026", false, 1, 1, []string{}},
		{"só título", "PORTARIA Nº 1/2026", true, 1, 1, []string{WarningTitleOnly}},
		{"dez páginas", "EDITAL", false, 3, 12, []string{WarningManyPages}},
		{"nove páginas", "EDITAL", false, 3, 11, []string{}},
		{"página desconhecida", "EDITAL", false, 0, 0, []string{}},
		{"tudo junto", "Designa:", true, 1, 40, []string{WarningNoNumber, WarningTitleOnly, WarningManyPages}},
	}
	for _, c := range cases {
		got := ActWarnings(c.title, c.titleOnly, c.start, c.end)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: esperava %v, veio %#v", c.name, c.want, got)
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
