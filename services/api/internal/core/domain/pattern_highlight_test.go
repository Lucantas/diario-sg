package domain

import "testing"

func TestPatternHighlightsTakesTheFirstFindingOfEachPatternWithFindings(t *testing.T) {
	reports := []PatternReport{
		{Pattern: Pattern{ID: "vazio"}},
		{Pattern: Pattern{ID: "a"}, Findings: []Finding{{Title: "a0"}, {Title: "a1"}, {Title: "a2"}}},
		{Pattern: Pattern{ID: "b"}, Findings: []Finding{{Title: "b0"}}},
		{Pattern: Pattern{ID: "c"}, Findings: []Finding{{Title: "c0"}}},
	}

	got := PatternHighlights(reports, 2)

	if len(got) != 2 || got[0].Pattern.ID != "a" || got[0].Finding.Title != "a0" || got[0].Findings != 3 || got[1].Pattern.ID != "b" {
		t.Fatalf("primeiro caso de cada padrão com caso, até o limite: %+v", got)
	}
}
