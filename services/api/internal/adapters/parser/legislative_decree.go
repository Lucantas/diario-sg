package parser

import (
	"regexp"
	"strings"
)

var legislativeDecreeLineRe = regexp.MustCompile(`(?i)^decreto\s+legislativo\s+n\s*[º°.]?\s*0*(\d+)\s*/\s*(\d{4})\.?\s*(.*)$`)

var signingDateRe = regexp.MustCompile(`^São Gonçalo,`)

func splitLegislativeDecreeTitles(lines []line) []line {
	out := make([]line, 0, len(lines))
	for _, l := range lines {
		m := legislativeDecreeLineRe.FindStringSubmatch(l.text)
		if m == nil || (m[3] != "" && !signingDateRe.MatchString(m[3])) {
			out = append(out, l)
			continue
		}
		out = append(out, line{text: "DECRETO LEGISLATIVO Nº " + m[1] + "/" + m[2], page: l.page})
		if rest := strings.TrimSpace(m[3]); rest != "" {
			out = append(out, line{text: rest, page: l.page})
		}
	}
	return out
}
