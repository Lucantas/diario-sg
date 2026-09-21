package postgres

import (
	"strings"
	"unicode"
)

const (
	startSel = "⟦"
	stopSel  = "⟧"
)

// highlightFallback destaca o termo quando o ts_headline não marcou nada:
// acontece quando o ato casou só pela substring (ILIKE), por exemplo um
// CNPJ ou um nome parcial. Compara sem acentos e sem caixa, como o banco.
func highlightFallback(snippet, query string) string {
	query = strings.TrimSpace(query)
	if query == "" || strings.Contains(snippet, startSel) {
		return snippet
	}
	folded := foldRunes(snippet)
	needle := foldRunes(query)
	idx := strings.Index(string(folded), string(needle))
	if idx < 0 {
		return snippet
	}
	start := len([]rune(string(folded)[:idx]))
	end := start + len(needle)
	r := []rune(snippet)
	if end > len(r) {
		return snippet
	}
	return string(r[:start]) + startSel + string(r[start:end]) + stopSel + string(r[end:])
}

var accentFold = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n',
}

// foldRunes devolve minúsculas sem acento, rune a rune (mesmo comprimento
// em runes que a entrada, para mapear posições de volta ao original).
func foldRunes(s string) []rune {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		r = unicode.ToLower(r)
		if f, ok := accentFold[r]; ok {
			r = f
		}
		out = append(out, r)
	}
	return out
}
