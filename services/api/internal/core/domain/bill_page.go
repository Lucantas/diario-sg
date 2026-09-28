package domain

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"golang.org/x/net/html"
)

var (
	saoPaulo        = mustLocation("America/Sao_Paulo")
	processKeyRe    = regexp.MustCompile(`(\d+)\s*/\s*(\d{4})`)
	docNumberRe     = regexp.MustCompile(`N\s*[º°oO]\.?\s*0*(\d+)\s*/\s*(\d{2,4})`)
	longDateRe      = regexp.MustCompile(`(\d{1,2}) DE ([A-ZÇÃÉÍÓÚÂÊÔ]+) DE (\d{4}) - (\d{2}):(\d{2})`)
	shortDateRe     = regexp.MustCompile(`(\d{2})/(\d{2})/(\d{4})(?:\s+às\s+(\d{2}):(\d{2}))?`)
	sitemapKeyRe    = regexp.MustCompile(`/areapublica/processo/(\d+)-(\d{4})$`)
	mainElementRe   = regexp.MustCompile(`(?s)<main\b.*?</main>`)
	portugueseMonth = map[string]time.Month{
		"janeiro": time.January, "fevereiro": time.February, "marco": time.March, "abril": time.April,
		"maio": time.May, "junho": time.June, "julho": time.July, "agosto": time.August,
		"setembro": time.September, "outubro": time.October, "novembro": time.November, "dezembro": time.December,
	}
)

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func BillPageMain(page []byte) []byte {
	return mainElementRe.Find(page)
}

func ParseBillPage(page []byte, base string) (Bill, error) {
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return Bill{}, err
	}
	hero := findFirst(doc, withClass("hero-card"))
	if hero == nil {
		return Bill{}, ErrBillNotFound
	}
	b := Bill{}
	if err := readHeroTitle(hero, &b); err != nil {
		return Bill{}, err
	}
	b.URL = strings.TrimRight(base, "/") + "/areapublica/processo/" + b.Key.Slug()
	if kind := findFirst(hero, withClass("fa-file-alt")); kind != nil && kind.Parent != nil {
		b.Kind = squeezed(nodeText(kind.Parent))
	}
	if box := findFirst(hero, withClass("ementa-box")); box != nil {
		b.Summary = squeezed(nodeText(box))
	}
	b.Authors = labeledValue(hero, "Autor(es):")
	b.PresentedOn = shortDate(labeledValue(hero, "Apresentação:"))
	readLaw(hero, base, &b)
	readSummary(doc, &b)
	b.Events = readEvents(doc)
	b.Opinions = readOpinions(doc)
	return b, nil
}

func readHeroTitle(hero *html.Node, b *Bill) error {
	title := findFirst(hero, withClass("hero-title"))
	if title == nil {
		return ErrBillNotFound
	}
	spans := findAll(title, withTag("span"))
	if len(spans) < 3 {
		return ErrBillNotFound
	}
	m := processKeyRe.FindStringSubmatch(nodeText(spans[len(spans)-1]))
	if m == nil {
		return ErrBillNotFound
	}
	b.Key.Number, _ = strconv.Atoi(m[1])
	b.Key.Year, _ = strconv.Atoi(m[2])
	b.DocLabel = squeezed(nodeText(spans[0]))
	if d := docNumberRe.FindStringSubmatch(b.DocLabel); d != nil {
		b.DocNumber, _ = strconv.Atoi(d[1])
		b.DocYear = fullYear(d[2])
	}
	return nil
}

func fullYear(s string) int {
	y, _ := strconv.Atoi(s)
	if len(s) == shortYearDigits {
		y += 2000
	}
	return y
}

func readLaw(hero *html.Node, base string, b *Bill) {
	link := findFirst(hero, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "a" && strings.Contains(firstAttr("", n, "href"), "documento/?Lei/")
	})
	if link == nil {
		return
	}
	m := processKeyRe.FindStringSubmatch(firstAttr("", link, "title") + " " + nodeText(link))
	if m == nil {
		return
	}
	b.LawNumber, _ = strconv.Atoi(m[1])
	b.LawYear, _ = strconv.Atoi(m[2])
	b.LawURL = strings.TrimRight(base, "/") + firstAttr("", link, "href")
}

func readSummary(doc *html.Node, b *Bill) {
	for _, box := range findAll(doc, withClass("summary-box")) {
		label := findFirst(box, withClass("summary-label"))
		value := findFirst(box, withClass("summary-value"))
		if label == nil || value == nil {
			continue
		}
		v := squeezed(nodeText(value))
		switch foldAccents(squeezed(nodeText(label))) {
		case "situacao atual":
			b.Status = v
		case "orgao / comissao atual":
			b.CurrentBody = v
		case "ultima movimentacao":
			b.LastMovement = v
		case "ultima atualizacao":
			b.SourceUpdatedAt = shortDateTime(v)
		}
	}
}

func readEvents(doc *html.Node) []BillEvent {
	var out []BillEvent
	for _, node := range findAll(doc, withClass("timeline-node")) {
		e := BillEvent{Position: len(out) + 1}
		if d := findFirst(node, withClass("timeline-date")); d != nil {
			if at := longDateTime(squeezed(nodeText(d))); at != nil {
				e.At = *at
			}
		}
		if badge := findFirst(node, withClass("badge")); badge != nil {
			e.Label = squeezed(nodeText(badge))
		}
		if title := findFirst(node, withClass("timeline-title")); title != nil {
			e.Text = squeezed(nodeText(title))
		}
		e.Sector = labeledText(node, "Setor:")
		out = append(out, e)
	}
	return out
}

func readOpinions(doc *html.Node) []BillOpinion {
	var out []BillOpinion
	for _, card := range findAll(doc, withClass("parecer-card")) {
		o := BillOpinion{Position: len(out) + 1}
		if badge := findFirst(card, withClass("badge")); badge != nil {
			o.Result = squeezed(nodeText(badge))
		}
		if day := findFirst(card, withClass("text-muted")); day != nil {
			o.On = shortDate(squeezed(nodeText(day)))
		}
		if title := findFirst(card, withClass("parecer-title")); title != nil {
			o.Committee = squeezed(nodeText(title))
		}
		o.Rapporteur = labeledText(card, "Relator(a):")
		out = append(out, o)
	}
	return out
}

func labeledValue(root *html.Node, label string) string {
	strong := findFirst(root, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "strong" && squeezed(nodeText(n)) == label
	})
	if strong == nil || strong.Parent == nil {
		return ""
	}
	for s := strong.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode && s.Data == "span" {
			return squeezed(nodeText(s))
		}
	}
	return ""
}

func labeledText(root *html.Node, label string) string {
	strong := findFirst(root, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "strong" && squeezed(nodeText(n)) == label
	})
	if strong == nil || strong.Parent == nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(squeezed(nodeText(strong.Parent)), label))
}

func longDateTime(s string) *time.Time {
	m := longDateRe.FindStringSubmatch(strings.ToUpper(s))
	if m == nil {
		return nil
	}
	month, ok := portugueseMonth[foldAccents(m[2])]
	if !ok {
		return nil
	}
	day, _ := strconv.Atoi(m[1])
	year, _ := strconv.Atoi(m[3])
	hour, _ := strconv.Atoi(m[4])
	minute, _ := strconv.Atoi(m[5])
	t := time.Date(year, month, day, hour, minute, 0, 0, saoPaulo).UTC()
	return &t
}

func shortDate(s string) *time.Time {
	m := shortDateRe.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	t, err := time.Parse("02/01/2006", m[1]+"/"+m[2]+"/"+m[3])
	if err != nil {
		return nil
	}
	return &t
}

func shortDateTime(s string) *time.Time {
	m := shortDateRe.FindStringSubmatch(s)
	if m == nil || m[4] == "" {
		return shortDate(s)
	}
	t, err := time.ParseInLocation("02/01/2006 15:04", m[1]+"/"+m[2]+"/"+m[3]+" "+m[4]+":"+m[5], saoPaulo)
	if err != nil {
		return nil
	}
	t = t.UTC()
	return &t
}

func withClass(class string) func(*html.Node) bool {
	return func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}
		for _, c := range strings.Fields(firstAttr("", n, "class")) {
			if c == class {
				return true
			}
		}
		return false
	}
}

func withTag(tag string) func(*html.Node) bool {
	return func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == tag }
}

func findFirst(root *html.Node, match func(*html.Node) bool) *html.Node {
	if match(root) {
		return root
	}
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if n := findFirst(c, match); n != nil {
			return n
		}
	}
	return nil
}

func findAll(root *html.Node, match func(*html.Node) bool) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if match(n) {
			out = append(out, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return out
}

type sitemapURLSet struct {
	URLs []struct {
		Loc string `xml:"loc"`
	} `xml:"url"`
	Sitemaps []struct {
		Loc string `xml:"loc"`
	} `xml:"sitemap"`
}

func ParseProcessSitemap(page []byte) ([]BillKey, error) {
	var set sitemapURLSet
	if err := xml.Unmarshal(page, &set); err != nil {
		return nil, fmt.Errorf("sitemap dos processos: %w", err)
	}
	var out []BillKey
	for _, u := range set.URLs {
		m := sitemapKeyRe.FindStringSubmatch(strings.TrimSpace(u.Loc))
		if m == nil {
			continue
		}
		number, _ := strconv.Atoi(m[1])
		year, _ := strconv.Atoi(m[2])
		out = append(out, BillKey{Number: number, Year: year})
	}
	return out, nil
}

func ParseSitemapIndex(page []byte, prefix string) ([]string, error) {
	var set sitemapURLSet
	if err := xml.Unmarshal(page, &set); err != nil {
		return nil, fmt.Errorf("índice do sitemap: %w", err)
	}
	var out []string
	for _, s := range set.Sitemaps {
		loc := strings.TrimSpace(s.Loc)
		if strings.Contains(loc, "/"+prefix) {
			out = append(out, loc)
		}
	}
	return out, nil
}

func InSaoPaulo(t time.Time) time.Time { return t.In(saoPaulo) }
