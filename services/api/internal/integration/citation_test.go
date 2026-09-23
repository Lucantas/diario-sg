//go:build integration

package integration

import (
	"strings"
	"testing"
)

type citableHits struct {
	Items []struct {
		Title     string `json:"title"`
		GazetteID string `json:"gazette_id"`
		PageStart *int   `json:"page_start"`
		PageEnd   *int   `json:"page_end"`
		PDFSHA256 string `json:"pdf_sha256"`
	} `json:"items"`
}

func TestSearchExposesPageAndPDFHash(t *testing.T) {
	srv, db := newOrganServerWithDB(t)
	if _, err := db.Exec(`UPDATE acts SET page_start = NULL, page_end = NULL WHERE position = 0`); err != nil {
		t.Fatal(err)
	}

	var hits citableHits
	getJSON(t, srv.URL+"/v1/acts", &hits)

	if len(hits.Items) != 3 {
		t.Fatalf("esperava 3 atos, veio %+v", hits)
	}
	for _, h := range hits.Items {
		if h.PDFSHA256 != strings.Repeat("e", 64) {
			t.Errorf("hash do PDF inesperado: %+v", h)
		}
		if strings.HasPrefix(h.Title, "DECRETO") {
			if h.PageStart != nil || h.PageEnd != nil {
				t.Errorf("ato sem página deve vir com null: %+v", h)
			}
			continue
		}
		if h.PageStart == nil || *h.PageStart != 1 || h.PageEnd == nil || *h.PageEnd != 1 {
			t.Errorf("esperava página 1: %+v", h)
		}
	}

	var gazette struct {
		PDFSHA256 string `json:"pdf_sha256"`
		Acts      []struct {
			PageStart *int   `json:"page_start"`
			Organ     string `json:"organ"`
			OrganName string `json:"organ_name"`
		} `json:"acts"`
	}
	getJSON(t, srv.URL+"/v1/gazettes/"+hits.Items[0].GazetteID, &gazette)
	if gazette.PDFSHA256 != strings.Repeat("e", 64) || len(gazette.Acts) != 3 ||
		gazette.Acts[0].PageStart != nil || gazette.Acts[1].PageStart == nil || gazette.Acts[1].Organ != "SEMAD" {
		t.Fatalf("edição inesperada: %+v", gazette)
	}
}
