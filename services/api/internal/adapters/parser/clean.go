package parser

import (
	"regexp"
	"strings"
)

var pageNoiseRe = regexp.MustCompile(`^(?:DIÁRIO OFICIAL(?: ELETRÔNICO DO MUNICÍPIO DE SÃO GONÇALO.*)?|PODER EXECUTIVO \(D\.O\.E\).*|\d{1,2} DE [A-ZÇ]+ DE \d{4} \| EDIÇÃO.*|D\.O\.E\. - \d{2}/\d{2}/\d{4}.*|_{10,})$`)

var (
	siteURLRe    = regexp.MustCompile(`^https://do\.pmsg\.rj\.gov\.br/?$`)
	pageNumberRe = regexp.MustCompile(`^\d{1,3}$`)
)

type line struct {
	text string
	page int
}

func joinTexts(ls []line, sep string) string {
	parts := make([]string, len(ls))
	for i, l := range ls {
		parts[i] = l.text
	}
	return strings.Join(parts, sep)
}

func splitLines(text string) []line {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out []line
	for p, page := range strings.Split(text, "\f") {
		pageLines := strings.Split(page, "\n")
		if n := len(pageLines); n > 0 && pageLines[n-1] == "" {
			pageLines = pageLines[:n-1]
		}
		for _, l := range pageLines {
			out = append(out, line{text: strings.TrimSpace(l), page: p + 1})
		}
	}
	return out
}

func (r Regex) stripSourceNoise(lines []line) []line {
	if r.pageHeader == nil {
		return lines
	}
	out := make([]line, 0, len(lines))
	headerPage := 0
	for _, l := range lines {
		if l.page != headerPage && (l.text == "" || r.pageHeader.MatchString(l.text)) {
			continue
		}
		headerPage = l.page
		if r.lineNoise.MatchString(l.text) {
			continue
		}
		out = append(out, l)
	}
	return out
}

func stripPageNoise(lines []line) []line {
	out := make([]line, 0, len(lines))
	for _, l := range lines {
		if pageNoiseRe.MatchString(l.text) {
			continue
		}
		if siteURLRe.MatchString(l.text) {
			if n := lastNonEmpty(out); n >= 0 && pageNumberRe.MatchString(out[n].text) {
				out = out[:n]
			}
			continue
		}
		out = append(out, l)
	}
	return out
}

func lastNonEmpty(lines []line) int {
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].text != "" {
			return i
		}
	}
	return -1
}

var preambleMarkers = map[string]bool{"ATOS DO PREFEITO": true, "GABINETE DO PREFEITO": true}

func dropPreamble(lines []line) []line {
	for i, l := range lines {
		if preambleMarkers[l.text] {
			return lines[i+1:]
		}
	}
	for i, l := range lines {
		if startsAct(l.text) {
			return lines[i:]
		}
	}
	return lines
}

var splitHeaderStart = map[string]bool{
	"TERMO": true, "EXTRATO": true, "AVISO": true, "EDITAL": true, "ATA": true,
	"CONTRATO": true, "NOTIFICAÇÃO": true, "RESOLUÇÃO": true, "PORTARIA": true,
	"DECRETO": true, "CHAMAMENTO": true, "CONCESSÃO": true, "DISPENSA": true,
	"INEXIGIBILIDADE": true, "PREGÃO": true,
}

var singleTokenRe = regexp.MustCompile(`^[A-ZÀ-Ü0-9º°.:/,()\-]+$`)

const maxJoinedTokens = 14

func joinSplitHeaders(lines []line) []line {
	out := make([]line, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if !splitHeaderStart[l.text] {
			out = append(out, l)
			continue
		}
		j := i + 1
		for j < len(lines) && j-i < maxJoinedTokens && singleTokenRe.MatchString(lines[j].text) && !splitHeaderStart[lines[j].text] {
			j++
		}
		if j-i < 3 {
			out = append(out, l)
			continue
		}
		out = append(out, line{text: joinTexts(lines[i:j], " "), page: l.page})
		i = j - 1
	}
	return out
}
