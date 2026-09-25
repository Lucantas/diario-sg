package domain

import (
	"sort"
	"time"
)

const (
	addendumLimitBP           = 2500
	renovationAddendumLimitBP = 5000
)

type AddendumAct struct {
	ActID        string
	ContractKeys []string
	Organ        string
	PublishedAt  time.Time
	Title        string
	Body         string
	IncreaseBP   int
}

type ExcessiveAddendum struct {
	ContractKey string
	Organ       string
	TotalBP     int
	LimitBP     int
	Acts        []AddendumAct
}

type addendumContract struct{ key, organ string }

func FindExcessiveAddenda(acts []AddendumAct) []ExcessiveAddendum {
	byContract := map[addendumContract][]AddendumAct{}
	for _, a := range acts {
		if a.IncreaseBP <= 0 {
			continue
		}
		for _, key := range a.ContractKeys {
			k := addendumContract{key, PrincipalOrgan(a.Organ)}
			byContract[k] = append(byContract[k], a)
		}
	}
	var out []ExcessiveAddendum
	for k, list := range byContract {
		if e, ok := excessiveAddendumOf(k, list); ok {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Acts[0].PublishedAt.Equal(out[j].Acts[0].PublishedAt) {
			return out[i].Acts[0].PublishedAt.Before(out[j].Acts[0].PublishedAt)
		}
		return out[i].ContractKey < out[j].ContractKey
	})
	return out
}

func excessiveAddendumOf(k addendumContract, list []AddendumAct) (ExcessiveAddendum, bool) {
	sort.Slice(list, func(i, j int) bool {
		if !list[i].PublishedAt.Equal(list[j].PublishedAt) {
			return list[i].PublishedAt.Before(list[j].PublishedAt)
		}
		return list[i].ActID < list[j].ActID
	})
	e := ExcessiveAddendum{ContractKey: k.key, Organ: k.organ, LimitBP: addendumLimitBP}
	seenOrdinal, seenIncrease := map[int]bool{}, map[int]bool{}
	for _, a := range list {
		if MentionsRenovation(a.Body) {
			e.LimitBP = renovationAddendumLimitBP
		}
		if ord := AddendumOrdinal(a.Title, a.Body); ord > 0 {
			if seenOrdinal[ord] {
				continue
			}
			seenOrdinal[ord] = true
		} else {
			if seenIncrease[a.IncreaseBP] {
				continue
			}
			seenIncrease[a.IncreaseBP] = true
		}
		e.TotalBP += a.IncreaseBP
		e.Acts = append(e.Acts, a)
	}
	return e, e.TotalBP > e.LimitBP
}
