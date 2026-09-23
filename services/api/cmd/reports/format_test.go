package main

import (
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestFormatReportLinksToThePageOfTheArchivedPDF(t *testing.T) {
	r := domain.ErrorReport{
		ID:            "r1",
		GazetteID:     "g1",
		Position:      3,
		ActTitle:      "PORTARIA Nº 12",
		Kind:          domain.ReportWrongOrgan,
		Message:       "órgão é a\nSEMAS",
		CreatedAt:     time.Date(2026, 9, 23, 15, 4, 0, 0, time.UTC),
		EditionNumber: "5.123",
		PublishedAt:   time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC),
		PageStart:     7,
	}

	got := formatReport(r, "https://diario.example/")

	want := "r1\t2026-09-23 12:04\tedição 5.123 de 22/09/2026, ato 3\torgao_errado\tPORTARIA Nº 12\tórgão é a SEMAS\thttps://diario.example/api/v1/gazettes/g1/pdf#page=7"
	if got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}

func TestFormatReportWithoutPageOrMessage(t *testing.T) {
	r := domain.ErrorReport{ID: "r2", GazetteID: "g2", Kind: domain.ReportOther}

	got := formatReport(r, "http://localhost:5173")

	if !strings.Contains(got, "\t(sem descrição)\t") {
		t.Fatalf("esperava marcação de descrição vazia: %q", got)
	}
	if !strings.HasSuffix(got, "\thttp://localhost:5173/api/v1/gazettes/g2/pdf") {
		t.Fatalf("esperava link sem página no fim: %q", got)
	}
}
