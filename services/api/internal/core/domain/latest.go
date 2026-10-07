package domain

import (
	"regexp"
	"sort"
	"time"
)

type LatestEdition struct {
	GazetteID     string
	Source        string
	Day           time.Time
	EditionNumber string
	IsExtra       bool
	Types         []TypeTotal
}

type TypeTotal struct {
	Type       ActType
	Acts       int
	ValueCents int64
}

var gazetteIDRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func ValidGazetteID(id string) bool { return gazetteIDRe.MatchString(id) }

func SortLatestEditions(editions []LatestEdition) {
	sort.SliceStable(editions, func(i, j int) bool {
		a, b := editions[i], editions[j]
		if sourceOrder(a.Source) != sourceOrder(b.Source) {
			return sourceOrder(a.Source) < sourceOrder(b.Source)
		}
		return !a.IsExtra && b.IsExtra
	})
	for _, e := range editions {
		sort.SliceStable(e.Types, func(i, j int) bool {
			if e.Types[i].Acts != e.Types[j].Acts {
				return e.Types[i].Acts > e.Types[j].Acts
			}
			return ActTypeName(e.Types[i].Type) < ActTypeName(e.Types[j].Type)
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
