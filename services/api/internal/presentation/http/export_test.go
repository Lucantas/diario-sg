package http

import (
	"reflect"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestCSVCellNeutralizesFormulas(t *testing.T) {
	cases := map[string]string{"=1+1": "'=1+1", "+55": "'+55", "-x": "'-x", "@SUM": "'@SUM", "\tx": "'\tx", "PORTARIA": "PORTARIA", "": ""}
	for in, want := range cases {
		if got := csvCell(in); got != want {
			t.Errorf("csvCell(%q) = %q, esperava %q", in, got, want)
		}
	}
}

func TestReaisBR(t *testing.T) {
	cases := map[int64]string{0: "0,00", 5: "0,05", 120000: "1200,00", 123456789: "1234567,89"}
	for in, want := range cases {
		if got := reaisBR(in); got != want {
			t.Errorf("reaisBR(%d) = %q, esperava %q", in, got, want)
		}
	}
}

func TestCSVRecord(t *testing.T) {
	h := domain.ActHit{Act: domain.Act{GazetteID: "g1", Type: domain.ActContrato, Title: "EXTRATO", Body: "=corpo",
		PageStart: 3, PageEnd: 4, Organ: "FMS"}, EditionNumber: "1771", PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
		IsExtra: true, SourceURL: "https://do/x.pdf", Checksum: "abc", CNPJs: []string{"1", "2"}, ValuesCents: []int64{120000, 30000}}

	got := csvRecord("https://site/", h)

	want := []string{"2026-09-18", "1771", "sim", "diario_prefeitura", "contrato", "FMS", domain.OrganName("FMS"), "EXTRATO", "3", "4",
		"1200,00 | 300,00", "1 | 2", "https://do/x.pdf#page=3", "https://site/api/v1/gazettes/g1/pdf#page=3", "abc", "", "'=corpo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("veio\n%q\nesperava\n%q", got, want)
	}
}

func TestCSVRecordWithoutPage(t *testing.T) {
	h := domain.ActHit{Act: domain.Act{GazetteID: "g1", Type: domain.ActOutro}, SourceURL: "https://do/x.pdf",
		PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)}

	got := csvRecord("https://site", h)

	if got[8] != "" || got[9] != "" || got[12] != "https://do/x.pdf" || got[13] != "https://site/api/v1/gazettes/g1/pdf" || got[2] != "não" {
		t.Errorf("veio %q", got)
	}

	h.Source = domain.SourceDiarioCamara
	if got = csvRecord("https://site", h); got[3] != "diario_camara" {
		t.Errorf("fonte da Câmara: %q", got[3])
	}
}
