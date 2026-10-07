package domain

type PatternHighlight struct {
	Pattern  Pattern
	Finding  Finding
	Findings int
}

func PatternHighlights(reports []PatternReport, limit int) []PatternHighlight {
	out := []PatternHighlight{}
	for _, r := range reports {
		if len(out) == limit {
			break
		}
		if len(r.Findings) > 0 {
			out = append(out, PatternHighlight{Pattern: r.Pattern, Finding: r.Findings[0], Findings: len(r.Findings)})
		}
	}
	return out
}
