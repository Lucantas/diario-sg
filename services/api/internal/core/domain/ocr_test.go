package domain

import (
	"slices"
	"testing"
)

func TestMarkReadByOCRFlagsTheActsOnOCRPages(t *testing.T) {
	acts := []Act{
		{Position: 1, PageStart: 1, PageEnd: 1},
		{Position: 2, PageStart: 2, PageEnd: 3},
		{Position: 3, PageStart: 4, PageEnd: 4},
		{Position: 4},
	}

	marked := MarkReadByOCR(acts, []int{3})

	var got []bool
	for _, a := range marked {
		got = append(got, a.ReadByOCR)
	}
	if !slices.Equal(got, []bool{false, true, false, false}) {
		t.Errorf("marcação: %v", got)
	}
	if acts[1].ReadByOCR {
		t.Error("alterou os atos de entrada")
	}
}

func TestReadByOCRBecomesAWarning(t *testing.T) {
	warnings := ActWarnings(WarningFactsOf(Act{Title: "DECRETO LEGISLATIVO 6/2020", Body: "texto", ReadByOCR: true}))

	if !slices.Contains(warnings, WarningReadByOCR) {
		t.Errorf("avisos: %v", warnings)
	}
}
