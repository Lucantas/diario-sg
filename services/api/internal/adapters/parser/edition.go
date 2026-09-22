package parser

import "regexp"

// O número aparece na capa ("EDIÇÃO N°1.771"; em 2025, às vezes "N°1. 362") e
// no cabeçalho de cada página ("| Nº 1.771 EM 18 DE SETEMBRO DE 2026"; em
// 2024, "N.º"; de 2020 a 2021, "| N.º 190 | em 08 de outubro de 2020",
// às vezes "em, 18 de agosto").
const editionDigits = `(\d{1,3}(?:\.\s?\d{3})+|\d+)`

var (
	editionNumberRes = []*regexp.Regexp{
		regexp.MustCompile(`EDIÇÃO N\s*[º°.]*\s*` + editionDigits),
		regexp.MustCompile(`(?i)\|\s*N\s*[º°.]*\s*` + editionDigits + `\s*\|?\s*EM,?\s+\d{1,2} DE`),
	}
	nonDigitRe = regexp.MustCompile(`\D`)
)

func (Regex) EditionNumber(text string) string {
	for _, re := range editionNumberRes {
		if m := re.FindStringSubmatch(text); m != nil {
			return nonDigitRe.ReplaceAllString(m[1], "")
		}
	}
	return ""
}
