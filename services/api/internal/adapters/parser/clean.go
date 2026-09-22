package parser

import (
	"regexp"
	"strings"
)

// Linhas fixas que o pdftotext gera em toda página do Diário (cabeçalho,
// rodapé com URL, rodapé do anexo de pessoal, separadores de anexo).
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

// splitLines normaliza quebras e remove espaços nas pontas de cada linha.
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

// stripPageNoise remove cabeçalho e rodapé de página. O número da página é
// uma linha só com dígitos imediatamente antes da URL do site; só é
// removido nesse contexto, porque tabelas também têm linhas só com dígitos.
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

// dropPreamble descarta capa (notícias) e expediente (lista de secretários):
// tudo antes de "ATOS DO PREFEITO" (edições até 2020: "GABINETE DO
// PREFEITO") ou, na falta dele, do primeiro início de ato.
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

// Palavras que iniciam um cabeçalho e que, quando o pdftotext quebra um
// título justificado em uma palavra por linha, precisam ser reunidas.
var splitHeaderStart = map[string]bool{
	"TERMO": true, "EXTRATO": true, "AVISO": true, "EDITAL": true, "ATA": true,
	"CONTRATO": true, "NOTIFICAÇÃO": true, "RESOLUÇÃO": true, "PORTARIA": true,
	"DECRETO": true, "CHAMAMENTO": true, "CONCESSÃO": true, "DISPENSA": true,
	"INEXIGIBILIDADE": true, "PREGÃO": true,
}

var singleTokenRe = regexp.MustCompile(`^[A-ZÀ-Ü0-9º°.:/,()\-]+$`)

const maxJoinedTokens = 14

// joinSplitHeaders reúne sequências como "TERMO / DE / APREENSÃO /
// ADMINISTRATIVA / Nº / 202/SEMMATRAN/2026" em uma linha só.
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
