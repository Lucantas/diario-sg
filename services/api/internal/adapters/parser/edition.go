package parser

import (
	"regexp"
	"strings"
)

// O número aparece na capa ("EDIÇÃO N°1.771") e no cabeçalho de cada página
// ("| Nº 1.771 EM 18 DE SETEMBRO DE 2026"; em 2024, "N.º").
var editionNumberRes = []*regexp.Regexp{
	regexp.MustCompile(`EDIÇÃO N\s*[º°.]*\s*([\d.]+)`),
	regexp.MustCompile(`\|\s*N\s*[º°.]*\s*([\d.]+)\s+EM\s+\d{1,2} DE`),
}

func (Regex) EditionNumber(text string) string {
	for _, re := range editionNumberRes {
		if m := re.FindStringSubmatch(text); m != nil {
			return strings.ReplaceAll(m[1], ".", "")
		}
	}
	return ""
}
