package http

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

var typeLabel = map[domain.ActType]string{
	domain.ActNomeacao: "Nomeação", domain.ActExoneracao: "Exoneração", domain.ActContrato: "Contrato",
	domain.ActAditivo: "Aditivo", domain.ActLicitacao: "Licitação", domain.ActDispensa: "Sem licitação",
	domain.ActDecreto: "Decreto", domain.ActLei: "Lei", domain.ActPortaria: "Portaria", domain.ActResolucao: "Resolução",
	domain.ActDespacho: "Despacho", domain.ActEdital: "Edital", domain.ActAta: "Ata", domain.ActCorrigenda: "Corrigenda",
	domain.ActPrestacaoContas: "Prestação de contas", domain.ActOutro: "Outro",
}

var saoPaulo = time.FixedZone("BRT", -3*60*60)

type rss struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	TTL         int       `xml:"ttl"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	GUID        rssGUID `xml:"guid"`
	PubDate     string  `xml:"pubDate"`
	Category    string  `xml:"category,omitempty"`
	Description string  `xml:"description"`
}

type rssGUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink string `xml:"isPermaLink,attr"`
}

func (a *API) actFeed(w http.ResponseWriter, r *http.Request) {
	f, ok := a.filterFromQuery(w, r)
	if !ok {
		return
	}
	hits, err := a.Feed.Execute(r.Context(), f)
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	title := "Diário SG: atos publicados"
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		title = "Diário SG: " + q
	}
	feed := rss{Version: "2.0", Channel: rssChannel{
		Title:       title,
		Link:        siteSearchURL(a.PublicWebURL, f),
		Description: "Atos mais recentes do Diário Oficial de São Gonçalo que casam com esta busca.",
		Language:    "pt-br",
		TTL:         15,
	}}
	for _, h := range hits {
		feed.Channel.Items = append(feed.Channel.Items, feedItem(a.PublicWebURL, h))
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=900")
	_, _ = io.WriteString(w, xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		a.Log.Warn("envio do feed interrompido", "error", err)
	}
}

func feedItem(base string, h domain.ActHit) rssItem {
	edition := h.EditionNumber
	if edition == "" {
		edition = "s/n"
	}
	return rssItem{
		Title:       typeLabel[h.Type] + ": " + h.Title,
		Link:        archivedURL(base, h),
		GUID:        rssGUID{Value: "diario-sg:" + h.GazetteID + ":" + strconv.Itoa(h.Position), IsPermaLink: "false"},
		PubDate:     time.Date(h.PublishedAt.Year(), h.PublishedAt.Month(), h.PublishedAt.Day(), 12, 0, 0, 0, saoPaulo).Format(time.RFC1123Z),
		Category:    h.Organ,
		Description: "Edição " + edition + ", " + h.PublishedAt.Format("02/01/2006") + ". " + stripMarks(h.Snippet),
	}
}

func stripMarks(s string) string {
	return strings.NewReplacer(startMark, "", stopMark, "").Replace(s)
}

const (
	startMark = "⟦"
	stopMark  = "⟧"
)

func siteSearchURL(base string, f domain.ActFilter) string {
	var params []string
	add := func(key, value string) {
		if value != "" {
			params = append(params, key+"="+url.QueryEscape(value))
		}
	}
	add("q", f.Query)
	add("tipo", string(f.Type))
	add("orgao", f.Organ)
	if !f.From.IsZero() {
		add("de", f.From.Format(time.DateOnly))
	}
	if !f.To.IsZero() {
		add("ate", f.To.Format(time.DateOnly))
	}
	if f.MinCents > 0 {
		add("valor_min", reaisBR(f.MinCents))
	}
	if f.MaxCents > 0 {
		add("valor_max", reaisBR(f.MaxCents))
	}
	link := strings.TrimRight(base, "/") + "/"
	if len(params) > 0 {
		link += "?" + strings.Join(params, "&")
	}
	return link
}
