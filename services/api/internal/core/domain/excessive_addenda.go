package domain

import (
	"sort"
	"time"
)

const (
	addendumRepublicationGap  = 30 * 24 * time.Hour
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
	for _, a := range withoutRepublications(acts) {
		if a.IncreaseBP <= 0 {
			continue
		}
		organ := PrincipalOrgan(a.Organ)
		if organ == "" {
			continue
		}
		for _, key := range a.ContractKeys {
			k := addendumContract{key, organ}
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
	seenOrdinal, seenIncrease, unnumberedIncrease := map[int]bool{}, map[int]bool{}, map[int]bool{}
	for _, a := range list {
		if MentionsRenovation(a.Title, a.Body) {
			e.LimitBP = renovationAddendumLimitBP
		}
		ord := AddendumOrdinal(a.Title, a.Body)
		repeated := seenIncrease[a.IncreaseBP]
		if ord > 0 {
			repeated = seenOrdinal[ord] || unnumberedIncrease[a.IncreaseBP]
		}
		if repeated {
			continue
		}
		seenOrdinal[ord] = ord > 0
		seenIncrease[a.IncreaseBP] = true
		unnumberedIncrease[a.IncreaseBP] = unnumberedIncrease[a.IncreaseBP] || ord == 0
		e.TotalBP += a.IncreaseBP
		e.Acts = append(e.Acts, a)
	}
	return e, e.TotalBP > e.LimitBP
}

type addendumIdentity struct {
	key      string
	ordinal  int
	increase int
}

func withoutRepublications(acts []AddendumAct) []AddendumAct {
	sorted := append([]AddendumAct(nil), acts...)
	sort.Slice(sorted, func(i, j int) bool {
		if !sorted[i].PublishedAt.Equal(sorted[j].PublishedAt) {
			return sorted[i].PublishedAt.Before(sorted[j].PublishedAt)
		}
		return sorted[i].ActID < sorted[j].ActID
	})
	firstSeen := map[addendumIdentity]time.Time{}
	var out []AddendumAct
	for _, a := range sorted {
		ord := AddendumOrdinal(a.Title, a.Body)
		republished := false
		for _, key := range a.ContractKeys {
			id := addendumIdentity{key, ord, a.IncreaseBP}
			if first, ok := firstSeen[id]; ok && a.PublishedAt.Sub(first) <= addendumRepublicationGap {
				republished = true
			} else if !ok {
				firstSeen[id] = a.PublishedAt
			}
		}
		if !republished {
			out = append(out, a)
		}
	}
	return out
}
