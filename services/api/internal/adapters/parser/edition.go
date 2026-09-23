package parser

import "regexp"

const editionDigits = `(\d{1,3}(?:\.\s?\d{3})+|\d+)`

var (
	editionNumberRes = []*regexp.Regexp{
		regexp.MustCompile(`EDIÇÃO N\s*[º°.]*\s*` + editionDigits),
		regexp.MustCompile(`(?i)\|\s*N\s*[º°.]*\s*` + editionDigits + `\s*\|?\s*EM,?\s+\d{1,2} DE`),
	}
	nonDigitRe = regexp.MustCompile(`\D`)
)

func (r Regex) EditionNumber(text string) string {
	for _, re := range r.editionRes {
		if m := re.FindStringSubmatch(text); m != nil {
			return nonDigitRe.ReplaceAllString(m[1], "")
		}
	}
	return ""
}
