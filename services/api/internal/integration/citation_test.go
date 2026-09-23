//go:build integration

package integration

import (
	"io"
	"net/http"
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

func TestArchivedPDFIsServedWithCacheValidators(t *testing.T) {
	srv, _ := newOrganServerWithDB(t)
	var hits citableHits
	getJSON(t, srv.URL+"/v1/acts", &hits)
	url := srv.URL + "/v1/gazettes/" + hits.Items[0].GazetteID + "/pdf"

	r, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	etag := `"` + strings.Repeat("e", 64) + `"`
	if r.StatusCode != http.StatusOK || string(body) != organGazette || r.Header.Get("Content-Type") != "application/pdf" ||
		r.Header.Get("ETag") != etag || !strings.Contains(r.Header.Get("Content-Disposition"), `filename="diario-sg-2026-09-18-7.pdf"`) {
		t.Fatalf("resposta inesperada: %d %v %q", r.StatusCode, r.Header, body)
	}

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("If-None-Match", etag)
	r, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(r.Body)
	r.Body.Close()
	if r.StatusCode != http.StatusNotModified || len(body) != 0 {
		t.Fatalf("esperava 304 sem corpo, veio %d %q", r.StatusCode, body)
	}

	r, err = http.Get(srv.URL + "/v1/gazettes/00000000-0000-0000-0000-000000000000/pdf")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusNotFound {
		t.Fatalf("edição inexistente deve dar 404, veio %d", r.StatusCode)
	}
}
