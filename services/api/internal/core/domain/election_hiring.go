package domain

import (
	"sort"
	"time"
)

var MunicipalElections = map[int]time.Time{2012: civilDate(2012, 10, 7), 2016: civilDate(2016, 10, 2), 2020: civilDate(2020, 11, 15), 2024: civilDate(2024, 10, 6)}

type MonthlyActCount struct {
	Type  ActType
	Year  int
	Month time.Month
	Count int
}

type HiringPeak struct {
	Type           ActType
	Year           int
	Month          time.Month
	Count          int
	BaselineMedian float64
	BaselineYears  int
	Election       time.Time
}

const (
	electionWindowMonths = 6
	electionPeakRatio    = 1.5
)

type monthKey struct {
	t ActType
	y int
	m time.Month
}

func FindElectionPeaks(counts []MonthlyActCount) []HiringPeak {
	byKey := map[monthKey]int{}
	types := map[ActType]bool{}
	for _, c := range counts {
		byKey[monthKey{c.Type, c.Year, c.Month}] = c.Count
		types[c.Type] = true
	}
	var out []HiringPeak
	for t := range types {
		for year, election := range MunicipalElections {
			first := civilDate(election.Year(), election.Month(), 1)
			for k := electionWindowMonths; k >= 1; k-- {
				day := first.AddDate(0, -k, 0)
				baseline := baselineCounts(byKey, t, day.Month())
				if len(baseline) == 0 {
					continue
				}
				median := medianOf(baseline)
				n := byKey[monthKey{t, day.Year(), day.Month()}]
				if n > 0 && float64(n) >= electionPeakRatio*median {
					out = append(out, HiringPeak{Type: t, Year: day.Year(), Month: day.Month(), Count: n,
						BaselineMedian: median, BaselineYears: len(baseline), Election: MunicipalElections[year]})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Year != out[j].Year {
			return out[i].Year < out[j].Year
		}
		if out[i].Month != out[j].Month {
			return out[i].Month < out[j].Month
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func baselineCounts(byKey map[monthKey]int, t ActType, m time.Month) []int {
	var out []int
	for k, n := range byKey {
		if _, election := MunicipalElections[k.y]; k.t == t && k.m == m && !election {
			out = append(out, n)
		}
	}
	return out
}

func medianOf(xs []int) float64 {
	s := append([]int(nil), xs...)
	sort.Ints(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return float64(s[mid])
	}
	return float64(s[mid-1]+s[mid]) / 2
}
