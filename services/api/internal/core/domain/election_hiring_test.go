package domain

import (
	"testing"
	"time"
)

func TestFindElectionPeaks(t *testing.T) {
	var counts []MonthlyActCount
	for _, y := range []int{2013, 2014, 2015, 2017, 2018} {
		counts = append(counts, MonthlyActCount{ActNomeacao, y, time.August, 100}, MonthlyActCount{ActNomeacao, y, time.November, 100})
	}
	counts = append(counts,
		MonthlyActCount{ActNomeacao, 2020, time.August, 150},
		MonthlyActCount{ActNomeacao, 2016, time.August, 149},
		MonthlyActCount{ActNomeacao, 2012, time.November, 400},
		MonthlyActCount{ActNomeacao, 2024, time.August, 1000},
		MonthlyActCount{ActNomeacao, 2024, time.November, 1000},
	)

	got := FindElectionPeaks(counts)

	if len(got) != 2 {
		t.Fatalf("esperava 2 picos (ago/2020 e ago/2024), veio %+v", got)
	}
	p := got[0]
	if p.Year != 2020 || p.Month != time.August || p.Count != 150 || p.BaselineMedian != 100 || p.BaselineYears != 5 || !p.Election.Equal(civilDate(2020, 11, 15)) {
		t.Errorf("pico inesperado: %+v", p)
	}
	if got[1].Year != 2024 || got[1].Month != time.August {
		t.Errorf("segundo pico inesperado: %+v", got[1])
	}
}

func TestFindElectionPeaksNeedsABaseline(t *testing.T) {
	counts := []MonthlyActCount{{ActExoneracao, 2020, time.August, 500}}

	if got := FindElectionPeaks(counts); len(got) != 0 {
		t.Fatalf("sem anos de comparação não há pico: %+v", got)
	}
}
