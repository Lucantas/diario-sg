package domain

import (
	"sort"
	"time"
)

type LatestDay struct {
	Source   string
	Day      time.Time
	Editions []LatestEdition
	Types    []TypeTotal
}

type LatestEdition struct {
	GazetteID     string
	EditionNumber string
	IsExtra       bool
}

type TypeTotal struct {
	Type       ActType
	Acts       int
	ValueCents int64
}

func SortLatestDays(days []LatestDay) {
	sort.SliceStable(days, func(i, j int) bool { return sourceOrder(days[i].Source) < sourceOrder(days[j].Source) })
	for _, d := range days {
		sort.SliceStable(d.Types, func(i, j int) bool {
			if d.Types[i].Acts != d.Types[j].Acts {
				return d.Types[i].Acts > d.Types[j].Acts
			}
			return ActTypeName(d.Types[i].Type) < ActTypeName(d.Types[j].Type)
		})
	}
}

func sourceOrder(source string) int {
	for i, s := range Sources {
		if s == source {
			return i
		}
	}
	return len(Sources)
}
