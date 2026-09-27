package domain

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	SourceMural         = "pmsg_mural"
	MuralTenders        = "licitacoes"
	MuralDirect         = "dispensas"
	MuralUnenforceable  = "inexigibilidades"
	MuralContracts      = "contratos"
	muralTenderColumns  = 6
	muralContractColumn = 6
	muralOpensLayout    = "02/01/2006 15:04"
	shortYearDigits     = 2
	yearDigits          = 4
	organNumberDigits   = 5
)

type Procurement struct {
	ID         int
	List       string
	Notice     string
	Process    string
	ProcessKey string
	Modality   string
	Criterion  string
	OpensAt    *time.Time
	Object     string
	Status     string
	URL        string
}

type ProcurementContract struct {
	ProcurementID int
	Notice        string
	Process       string
	ProcessKey    string
	Modality      string
	Object        string
	ValueCents    int64
	Supplier      string
	Instrument    string
	DocumentURL   string
}

func ProcessKey(process string) string {
	number, year, ok := strings.Cut(strings.TrimSpace(process), "/")
	year, _, _ = strings.Cut(strings.TrimSpace(year), "-")
	if len(year) == shortYearDigits {
		year = "20" + year
	}
	number = strings.TrimSpace(number)
	if prefix, rest, dotted := strings.Cut(number, "."); dotted && len(rest) >= organNumberDigits && allDigits(prefix) {
		number = rest
	}
	number = strings.TrimLeft(strings.ReplaceAll(number, ".", ""), "0")
	if !ok || !allDigits(number) || len(year) != yearDigits || !allDigits(year) {
		return ""
	}
	return number + "/" + year
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

type muralCell struct {
	text, strong, small, href string
}

func ParseProcurements(list string, page []byte, base string) ([]Procurement, error) {
	rows, err := muralRows(page, muralTenderColumns)
	if err != nil {
		return nil, fmt.Errorf("lista %s do mural: %w", list, err)
	}
	out := make([]Procurement, 0, len(rows))
	for _, c := range rows {
		id, err := muralID(c[0].href)
		if err != nil {
			return nil, fmt.Errorf("lista %s do mural: %w", list, err)
		}
		p := Procurement{ID: id, List: list, Notice: c[0].strong, Process: c[0].small, ProcessKey: ProcessKey(c[0].small),
			Modality: c[1].strong, Criterion: c[1].small, Object: c[3].text, Status: c[4].text,
			URL: base + strings.TrimPrefix(c[0].href, "./")}
		if t, err := time.Parse(muralOpensLayout, c[2].text); err == nil {
			p.OpensAt = &t
		}
		out = append(out, p)
	}
	return out, nil
}

func ParseProcurementContracts(page []byte, base string) ([]ProcurementContract, error) {
	rows, err := muralRows(page, muralContractColumn)
	if err != nil {
		return nil, fmt.Errorf("contratos do mural: %w", err)
	}
	out := make([]ProcurementContract, 0, len(rows))
	for _, c := range rows {
		id, err := muralID(c[0].href)
		if err != nil {
			return nil, fmt.Errorf("contratos do mural: %w", err)
		}
		value, err := portalCents(c[3].text)
		if err != nil {
			return nil, fmt.Errorf("contrato da licitação %d: valor %q: %w", id, c[3].text, err)
		}
		pc := ProcurementContract{ProcurementID: id, Notice: c[0].strong, Process: c[0].small, ProcessKey: ProcessKey(c[0].small),
			Modality: c[1].strong, Object: c[2].text, ValueCents: value, Supplier: c[4].text, Instrument: c[5].text}
		if c[5].href != "" {
			pc.DocumentURL = base + strings.TrimPrefix(c[5].href, "./")
		}
		out = append(out, pc)
	}
	return out, nil
}

func muralID(href string) (int, error) {
	u, err := url.Parse(href)
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(u.Query().Get("licitacao_id"))
	if err != nil {
		return 0, fmt.Errorf("link %q sem licitacao_id", href)
	}
	return id, nil
}

func muralRows(page []byte, columns int) ([][]muralCell, error) {
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return nil, err
	}
	var rows [][]muralCell
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			if cells := muralCells(n); len(cells) > 0 {
				rows = append(rows, cells)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(rows) == 0 {
		return nil, fmt.Errorf("nenhuma linha na tabela")
	}
	for _, r := range rows {
		if len(r) != columns {
			return nil, fmt.Errorf("linha com %d colunas, esperava %d", len(r), columns)
		}
	}
	return rows, nil
}

func muralCells(tr *html.Node) []muralCell {
	var cells []muralCell
	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		if td.Type != html.ElementNode || td.Data != "td" {
			continue
		}
		c := muralCell{text: squeezed(nodeText(td))}
		var find func(*html.Node)
		find = func(n *html.Node) {
			if n.Type == html.ElementNode {
				switch n.Data {
				case "strong":
					if c.strong == "" {
						c.strong = squeezed(nodeText(n))
					}
				case "small":
					if c.small == "" {
						c.small = squeezed(nodeText(n))
					}
				case "a":
					for _, a := range n.Attr {
						if a.Key == "href" && c.href == "" {
							c.href = a.Val
						}
					}
				}
			}
			for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
				find(ch)
			}
		}
		find(td)
		cells = append(cells, c)
	}
	return cells
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

type MuralMatches struct {
	Procurements []Procurement
	Contracts    []ProcurementContract
}
