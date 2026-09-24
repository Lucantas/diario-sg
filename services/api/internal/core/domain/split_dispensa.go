package domain

import (
	"sort"
	"time"
)

type DispensaProcess struct{ Key, Label string }

type DispensaAct struct {
	ActID       string
	CNPJ        string
	OtherCNPJs  []string
	Processes   []DispensaProcess
	Organ       string
	PublishedAt time.Time
	ValueCents  int64
	Body        string
}

type DispensaContract struct {
	Processes      []DispensaProcess
	FirstPublished time.Time
	ValueCents     int64
	LimitCents     int64
	Organs         []string
	ActIDs         []string
}

type SplitDispensa struct {
	CNPJ       string
	Year       int
	Contracts  []DispensaContract
	TotalCents int64
	LimitCents int64
}

func FindSplitDispensas(acts []DispensaAct) []SplitDispensa {
	byCNPJ := map[string][]DispensaAct{}
	for _, a := range acts {
		if _, public := PublicBody(a.CNPJ); public || sharedWithSupplier(a) || CitesEmergency(a.Body) || a.ValueCents <= 0 {
			continue
		}
		byCNPJ[a.CNPJ] = append(byCNPJ[a.CNPJ], a)
	}
	var out []SplitDispensa
	for cnpj, list := range byCNPJ {
		byYear := map[int][]DispensaContract{}
		for _, c := range valueDispensaContracts(list) {
			byYear[c.FirstPublished.Year()] = append(byYear[c.FirstPublished.Year()], c)
		}
		for year, contracts := range byYear {
			if s, ok := splitOf(cnpj, year, contracts); ok {
				out = append(out, s)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Year != out[j].Year {
			return out[i].Year < out[j].Year
		}
		return out[i].CNPJ < out[j].CNPJ
	})
	return out
}

func splitOf(cnpj string, year int, contracts []DispensaContract) (SplitDispensa, bool) {
	s := SplitDispensa{CNPJ: cnpj, Year: year, Contracts: contracts}
	for _, c := range contracts {
		s.TotalCents += c.ValueCents
		s.LimitCents = max(s.LimitCents, c.LimitCents)
	}
	sort.Slice(s.Contracts, func(i, j int) bool { return s.Contracts[i].FirstPublished.Before(s.Contracts[j].FirstPublished) })
	return s, len(contracts) >= 2 && s.TotalCents > s.LimitCents
}

func valueDispensaContracts(acts []DispensaAct) []DispensaContract {
	sort.Slice(acts, func(i, j int) bool {
		if !acts[i].PublishedAt.Equal(acts[j].PublishedAt) {
			return acts[i].PublishedAt.Before(acts[j].PublishedAt)
		}
		return acts[i].ActID < acts[j].ActID
	})
	groups := newUnionFind(len(acts))
	firstByProcess := map[string]int{}
	for i, a := range acts {
		for _, p := range a.Processes {
			if j, ok := firstByProcess[p.Key]; ok {
				groups.union(i, j)
			} else {
				firstByProcess[p.Key] = i
			}
		}
	}
	for i, a := range acts {
		if len(a.Processes) > 0 && (!IsRepublication(a.Body) || groups.size(i) > 1) {
			continue
		}
		if j, ok := sameValueSameYear(acts, i, groups); ok {
			groups.union(i, j)
		}
	}
	var out []DispensaContract
	for _, members := range groups.sets() {
		if c, ok := contractOf(acts, members); ok {
			out = append(out, c)
		}
	}
	return out
}

func sameValueSameYear(acts []DispensaAct, i int, groups *unionFind) (int, bool) {
	a := acts[i]
	for j, b := range acts {
		if groups.find(j) != groups.find(i) && b.ValueCents == a.ValueCents && b.PublishedAt.Year() == a.PublishedAt.Year() {
			return j, true
		}
	}
	return 0, false
}

func contractOf(acts []DispensaAct, members []int) (DispensaContract, bool) {
	var c DispensaContract
	valueDispensa, citesLei14133 := false, false
	seenProcess, seenOrgan := map[string]bool{}, map[string]bool{}
	for _, i := range members {
		a := acts[i]
		if c.FirstPublished.IsZero() || a.PublishedAt.Before(c.FirstPublished) {
			c.FirstPublished = a.PublishedAt
		}
		c.ValueCents = max(c.ValueCents, a.ValueCents)
		citesLei14133 = citesLei14133 || CitesLei14133(a.Body)
		valueDispensa = valueDispensa || CitesValueDispensa(a.Body)
		c.ActIDs = append(c.ActIDs, a.ActID)
		for _, p := range a.Processes {
			if !seenProcess[p.Key] {
				seenProcess[p.Key] = true
				c.Processes = append(c.Processes, p)
			}
		}
		if a.Organ != "" && !seenOrgan[a.Organ] {
			seenOrgan[a.Organ] = true
			c.Organs = append(c.Organs, a.Organ)
		}
	}
	c.LimitCents = DispensaLimitCents(c.FirstPublished, citesLei14133)
	return c, valueDispensa && c.ValueCents < c.LimitCents
}

func sharedWithSupplier(a DispensaAct) bool {
	for _, other := range a.OtherCNPJs {
		if _, public := PublicBody(other); !public {
			return true
		}
	}
	return false
}

type unionFind struct{ parent []int }

func newUnionFind(n int) *unionFind {
	u := &unionFind{parent: make([]int, n)}
	for i := range u.parent {
		u.parent[i] = i
	}
	return u
}

func (u *unionFind) find(i int) int {
	for u.parent[i] != i {
		u.parent[i] = u.parent[u.parent[i]]
		i = u.parent[i]
	}
	return i
}

func (u *unionFind) union(i, j int) {
	ri, rj := u.find(i), u.find(j)
	if ri < rj {
		u.parent[rj] = ri
	} else if rj < ri {
		u.parent[ri] = rj
	}
}

func (u *unionFind) size(i int) int {
	n, root := 0, u.find(i)
	for j := range u.parent {
		if u.find(j) == root {
			n++
		}
	}
	return n
}

func (u *unionFind) sets() [][]int {
	byRoot := map[int][]int{}
	var roots []int
	for i := range u.parent {
		r := u.find(i)
		if _, ok := byRoot[r]; !ok {
			roots = append(roots, r)
		}
		byRoot[r] = append(byRoot[r], i)
	}
	out := make([][]int, 0, len(roots))
	for _, r := range roots {
		out = append(out, byRoot[r])
	}
	return out
}
