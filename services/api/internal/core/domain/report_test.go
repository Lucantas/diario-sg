package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewErrorReport(t *testing.T) {
	r, err := NewErrorReport("g1", 3, "  PORTARIA Nº 5/2026 ", ReportWrongType, "  era nomeação  ")
	if err != nil {
		t.Fatal(err)
	}
	if r.ActTitle != "PORTARIA Nº 5/2026" || r.Message != "era nomeação" || r.Status != ReportOpen {
		t.Errorf("normalização inesperada: %+v", r)
	}
	if _, err := NewErrorReport("g1", 0, "T", ReportWrongPage, ""); err != nil {
		t.Errorf("posição zero é válida: %v", err)
	}
}

func TestNewErrorReportRejectsInvalidInput(t *testing.T) {
	cases := map[string]struct {
		gazette  string
		position int
		title    string
		kind     ReportKind
		message  string
	}{
		"sem edição":        {"", 1, "T", ReportWrongText, ""},
		"posição negativa":  {"g1", -1, "T", ReportWrongText, ""},
		"sem título":        {"g1", 1, "  ", ReportWrongText, ""},
		"título longo":      {"g1", 1, strings.Repeat("a", 501), ReportWrongText, ""},
		"tipo desconhecido": {"g1", 1, "T", "bobagem", "x"},
		"outro sem texto":   {"g1", 1, "T", ReportOther, "   "},
		"mensagem longa":    {"g1", 1, "T", ReportWrongText, strings.Repeat("a", 2001)},
	}
	for name, c := range cases {
		if _, err := NewErrorReport(c.gazette, c.position, c.title, c.kind, c.message); !errors.Is(err, ErrInvalidReport) {
			t.Errorf("%s: esperava ErrInvalidReport, veio %v", name, err)
		}
	}
}

func TestReportStatusValidForClosing(t *testing.T) {
	if !ReportResolved.ClosesReport() || !ReportDiscarded.ClosesReport() || ReportOpen.ClosesReport() {
		t.Error("só resolvido e descartado fecham um reporte")
	}
}
