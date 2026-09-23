//go:build integration

package integration

import (
	"encoding/xml"
	"net/http"
	"strings"
	"testing"
)

type feedDoc struct {
	Channel struct {
		Title string `xml:"title"`
		Link  string `xml:"link"`
		Items []struct {
			Title string `xml:"title"`
			Link  string `xml:"link"`
			GUID  string `xml:"guid"`
		} `xml:"item"`
	} `xml:"channel"`
}

func TestFeedListsRecentActsOfTheSearch(t *testing.T) {
	srv := newInvestigatorServer(t)

	r, body := fetch(t, srv.URL+"/v1/feeds/acts?type=contrato")

	var doc feedDoc
	if err := xml.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("XML inválido: %v\n%s", err, body)
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/rss+xml") || r.Header.Get("Cache-Control") == "" {
		t.Fatalf("cabeçalhos inesperados: %v", r.Header)
	}
	items := doc.Channel.Items
	if len(items) != 3 || doc.Channel.Link != "https://web.exemplo/?tipo=contrato" || doc.Channel.Title != "Diário SG: atos publicados" {
		t.Fatalf("canal inesperado: %+v", doc.Channel)
	}
	for i, it := range items {
		if !strings.HasPrefix(it.GUID, "diario-sg:") || !strings.HasSuffix(it.GUID, ":"+string(rune('0'+i))) ||
			!strings.HasPrefix(it.Link, "https://web.exemplo/api/v1/gazettes/") || !strings.HasSuffix(it.Link, "/pdf#page=1") ||
			!strings.HasPrefix(it.Title, "Contrato: EXTRATO DO CONTRATO") {
			t.Fatalf("item %d inesperado: %+v", i, it)
		}
	}

	_, body = fetch(t, srv.URL+"/v1/feeds/acts?q=limpeza&min_value=40000")
	doc = feedDoc{}
	if err := xml.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Channel.Items) != 1 || doc.Channel.Title != "Diário SG: limpeza" {
		t.Fatalf("feed filtrado inesperado: %+v", doc.Channel)
	}

	r, _ = fetch(t, srv.URL+"/v1/feeds/acts?organ=TOTAL")
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("filtro inválido deve dar 400, veio %d", r.StatusCode)
	}
}
