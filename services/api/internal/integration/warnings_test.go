//go:build integration

package integration

import (
	"reflect"
	"strings"
	"testing"
)

const warningsGazette = "Nomeia:\nFULANO DE TAL para o cargo de Assessor.\n" +
	"PORTARIA Nº 5/2026\n" +
	"DECRETO Nº 9/2026\nDispõe sobre o horário das repartições."

type warnedHits struct {
	Items []struct {
		Title     string   `json:"title"`
		GazetteID string   `json:"gazette_id"`
		Position  *int     `json:"position"`
		Warnings  []string `json:"warnings"`
	} `json:"items"`
}

func TestWarningsInSearchGazetteAndExport(t *testing.T) {
	srv, _ := newServerFor(t, warningsGazette)

	var hits warnedHits
	getJSON(t, srv.URL+"/v1/acts", &hits)

	got := map[string][]string{}
	for _, h := range hits.Items {
		got[h.Title] = h.Warnings
	}
	want := map[string][]string{
		"Nomeia:":            {"sem_numero"},
		"PORTARIA Nº 5/2026": {"so_titulo"},
		"DECRETO Nº 9/2026":  {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("avisos na busca: esperava %v, veio %v", want, got)
	}

	positions := map[string]int{}
	for _, h := range hits.Items {
		if h.Position == nil {
			t.Fatalf("busca sem a posição do ato, que o reporte de erro precisa: %s", h.Title)
		}
		positions[h.Title] = *h.Position
	}
	if want := map[string]int{"Nomeia:": 0, "PORTARIA Nº 5/2026": 1, "DECRETO Nº 9/2026": 2}; !reflect.DeepEqual(positions, want) {
		t.Fatalf("posições na busca: esperava %v, veio %v", want, positions)
	}

	var gazette struct {
		Acts []struct {
			Title    string   `json:"title"`
			Warnings []string `json:"warnings"`
		} `json:"acts"`
	}
	getJSON(t, srv.URL+"/v1/gazettes/"+hits.Items[0].GazetteID, &gazette)
	for _, a := range gazette.Acts {
		if !reflect.DeepEqual(a.Warnings, want[a.Title]) {
			t.Fatalf("avisos na edição: %s veio %v", a.Title, a.Warnings)
		}
	}

	_, body := fetch(t, srv.URL+"/v1/acts/export")
	records := readCSV(t, body)
	header := strings.Join(records[0], ";")
	if !strings.Contains(header, ";avisos;texto") {
		t.Fatalf("CSV sem a coluna avisos: %s", header)
	}
	found := false
	for _, r := range records[1:] {
		if r[7] == "PORTARIA Nº 5/2026" && r[15] == "so_titulo" {
			found = true
		}
	}
	if !found {
		t.Fatalf("CSV sem o aviso da portaria só com título: %q", records)
	}
}
