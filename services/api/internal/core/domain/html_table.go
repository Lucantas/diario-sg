package domain

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

type tableCell struct {
	text, strong, small, href, onclick string
}

func muralRows(page []byte, columns int) ([][]tableCell, error) {
	rows, err := tableRows(page)
	if err != nil {
		return nil, err
	}
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

func tableRows(page []byte) ([][]tableCell, error) {
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return nil, err
	}
	var rows [][]tableCell
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			if cells := tableCells(n); len(cells) > 0 {
				rows = append(rows, cells)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return rows, nil
}

func tableCells(tr *html.Node) []tableCell {
	var cells []tableCell
	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		if td.Type != html.ElementNode || td.Data != "td" {
			continue
		}
		c := tableCell{text: squeezed(nodeText(td))}
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
					c.href = firstAttr(c.href, n, "href")
				case "button":
					c.onclick = firstAttr(c.onclick, n, "onclick")
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

func firstAttr(current string, n *html.Node, key string) string {
	if current != "" {
		return current
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
