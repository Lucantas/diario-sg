package http

import (
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestSiteSearchURL(t *testing.T) {
	f := domain.ActFilter{Query: "limpeza or coleta", Type: domain.ActContrato, Organ: "SEMED",
		From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		MinCents: 150050}

	got := siteSearchURL("https://site/", f)

	want := "https://site/?q=limpeza+or+coleta&tipo=contrato&orgao=SEMED&de=2024-01-01&ate=2024-12-31&valor_min=1500%2C50"
	if got != want {
		t.Errorf("veio\n%s\nesperava\n%s", got, want)
	}
	if got := siteSearchURL("https://site", domain.ActFilter{}); got != "https://site/" {
		t.Errorf("busca vazia deveria ser a página inicial, veio %s", got)
	}
}

func TestStripMarks(t *testing.T) {
	if got := stripMarks("a ⟦b⟧ c"); got != "a b c" {
		t.Errorf("veio %q", got)
	}
}

func TestFeedItem(t *testing.T) {
	h := domain.ActHit{Act: domain.Act{GazetteID: "g1", Type: domain.ActContrato, Title: "EXTRATO", Position: 7,
		PageStart: 3, Organ: "FMS"}, PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
		EditionNumber: "1771", Snippet: "valor ⟦global⟧"}

	it := feedItem("https://site", h)

	if it.Title != "Contrato: EXTRATO" || it.Link != "https://site/api/v1/gazettes/g1/pdf#page=3" ||
		it.GUID.Value != "diario-sg:g1:7" || it.GUID.IsPermaLink != "false" || it.Category != "FMS" ||
		it.PubDate != "Fri, 18 Sep 2026 12:00:00 -0300" || it.Description != "Edição 1771, 18/09/2026. valor global" {
		t.Errorf("item inesperado: %+v", it)
	}

	h.Organ, h.PageStart, h.EditionNumber = "", 0, ""
	it = feedItem("https://site", h)
	if it.Category != "" || it.Link != "https://site/api/v1/gazettes/g1/pdf" || it.Description != "Edição s/n, 18/09/2026. valor global" {
		t.Errorf("item sem órgão e página inesperado: %+v", it)
	}
}
