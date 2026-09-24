package domain

import (
	"slices"
	"sort"
	"strconv"
	"time"
)

const SupplierPanelLimit = 50

type PanelAct struct {
	ActID       string
	CNPJ        string
	OtherCNPJs  []string
	Refs        []string
	Organ       string
	PublishedAt time.Time
	ValueCents  int64
	Type        ActType
	Title       string
	Head        string
}

type PanelFilter struct {
	Year  int
	Organ string
}

type SupplierRow struct {
	CNPJ            string
	Contracts       int
	ContractedCents int64
	RegisteredCents int64
	First           time.Time
	Last            time.Time
	Organs          []string
	LargestActID    string
	largest         panelContract
}

type PanelTotal struct {
	Key             string
	Contracts       int
	ContractedCents int64
	RegisteredCents int64
}

type SupplierPanel struct {
	Filter          PanelFilter
	Contracts       int
	Suppliers       int
	ContractedCents int64
	RegisteredCents int64
	Rows            []SupplierRow
	Years           []PanelTotal
	Organs          []PanelTotal
}

type panelContract struct {
	CNPJ            string
	First           time.Time
	Last            time.Time
	ContractedCents int64
	RegisteredCents int64
	Organs          []string
	LargestActID    string
}

func (c panelContract) outweighs(o panelContract) bool {
	if c.ContractedCents != o.ContractedCents {
		return c.ContractedCents > o.ContractedCents
	}
	return c.RegisteredCents > o.RegisteredCents
}

func (f PanelFilter) matchesYear(c panelContract) bool {
	return f.Year == 0 || c.First.Year() == f.Year
}

func (f PanelFilter) matchesOrgan(c panelContract) bool {
	return f.Organ == "" || slices.Contains(c.Organs, PrincipalOrgan(f.Organ))
}

func BuildSupplierPanel(acts []PanelAct, f PanelFilter) SupplierPanel {
	contracts := panelContracts(acts)
	p := SupplierPanel{Filter: f, Rows: []SupplierRow{}, Years: yearTotals(contracts, f), Organs: organTotals(contracts, f)}
	rows := map[string]*SupplierRow{}
	for _, c := range contracts {
		if !f.matchesYear(c) || !f.matchesOrgan(c) {
			continue
		}
		p.Contracts++
		p.ContractedCents += c.ContractedCents
		p.RegisteredCents += c.RegisteredCents
		row, ok := rows[c.CNPJ]
		if !ok {
			row = &SupplierRow{CNPJ: c.CNPJ, First: c.First, Last: c.Last}
			rows[c.CNPJ] = row
		}
		addToRow(row, c)
	}
	for _, row := range rows {
		sort.Strings(row.Organs)
		p.Rows = append(p.Rows, *row)
	}
	sort.Slice(p.Rows, func(i, j int) bool {
		a, b := p.Rows[i], p.Rows[j]
		if a.ContractedCents != b.ContractedCents {
			return a.ContractedCents > b.ContractedCents
		}
		if a.RegisteredCents != b.RegisteredCents {
			return a.RegisteredCents > b.RegisteredCents
		}
		return a.CNPJ < b.CNPJ
	})
	p.Suppliers = len(p.Rows)
	p.Rows = p.Rows[:min(len(p.Rows), SupplierPanelLimit)]
	return p
}

func addToRow(row *SupplierRow, c panelContract) {
	row.Contracts++
	row.ContractedCents += c.ContractedCents
	row.RegisteredCents += c.RegisteredCents
	if c.First.Before(row.First) {
		row.First = c.First
	}
	if c.Last.After(row.Last) {
		row.Last = c.Last
	}
	for _, o := range c.Organs {
		if !slices.Contains(row.Organs, o) {
			row.Organs = append(row.Organs, o)
		}
	}
	if row.LargestActID == "" || c.outweighs(row.largest) {
		row.largest, row.LargestActID = c, c.LargestActID
	}
}

func yearTotals(contracts []panelContract, f PanelFilter) []PanelTotal {
	byYear := map[string]*PanelTotal{}
	for _, c := range contracts {
		if f.matchesOrgan(c) {
			addToTotal(byYear, strconv.Itoa(c.First.Year()), c)
		}
	}
	out := totalsOf(byYear)
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func organTotals(contracts []panelContract, f PanelFilter) []PanelTotal {
	byOrgan := map[string]*PanelTotal{}
	for _, c := range contracts {
		if !f.matchesYear(c) {
			continue
		}
		for _, o := range c.Organs {
			addToTotal(byOrgan, o, c)
		}
	}
	out := totalsOf(byOrgan)
	sort.Slice(out, func(i, j int) bool {
		if out[i].ContractedCents != out[j].ContractedCents {
			return out[i].ContractedCents > out[j].ContractedCents
		}
		if out[i].RegisteredCents != out[j].RegisteredCents {
			return out[i].RegisteredCents > out[j].RegisteredCents
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func addToTotal(totals map[string]*PanelTotal, key string, c panelContract) {
	t, ok := totals[key]
	if !ok {
		t = &PanelTotal{Key: key}
		totals[key] = t
	}
	t.Contracts++
	t.ContractedCents += c.ContractedCents
	t.RegisteredCents += c.RegisteredCents
}

func totalsOf(totals map[string]*PanelTotal) []PanelTotal {
	out := make([]PanelTotal, 0, len(totals))
	for _, t := range totals {
		out = append(out, *t)
	}
	return out
}

func panelContracts(acts []PanelAct) []panelContract {
	bySupplier := map[string][]PanelAct{}
	for _, a := range acts {
		if _, public := PublicBody(a.CNPJ); public || a.ValueCents <= 0 || citesAnotherSupplier(a.OtherCNPJs) {
			continue
		}
		bySupplier[a.CNPJ] = append(bySupplier[a.CNPJ], a)
	}
	var out []panelContract
	for _, list := range bySupplier {
		sortPanelActs(list)
		for _, members := range joinPanelActs(list).sets() {
			if c, ok := panelContractOf(list, members); ok {
				out = append(out, c)
			}
		}
	}
	return out
}

func sortPanelActs(acts []PanelAct) {
	sort.Slice(acts, func(i, j int) bool {
		if !acts[i].PublishedAt.Equal(acts[j].PublishedAt) {
			return acts[i].PublishedAt.Before(acts[j].PublishedAt)
		}
		return acts[i].ActID < acts[j].ActID
	})
}

func joinPanelActs(acts []PanelAct) *unionFind {
	groups := newUnionFind(len(acts))
	firstByRef := map[string]int{}
	for i, a := range acts {
		for _, ref := range a.Refs {
			if j, ok := firstByRef[ref]; ok {
				groups.union(i, j)
			} else {
				firstByRef[ref] = i
			}
		}
	}
	for i, a := range acts {
		if len(a.Refs) > 0 {
			continue
		}
		for j, b := range acts {
			if groups.find(j) != groups.find(i) && b.ValueCents == a.ValueCents && b.PublishedAt.Year() == a.PublishedAt.Year() {
				groups.union(i, j)
				break
			}
		}
	}
	return groups
}

func panelContractOf(acts []PanelAct, members []int) (panelContract, bool) {
	c := panelContract{CNPJ: acts[members[0]].CNPJ, First: acts[members[0]].PublishedAt}
	var registeredID string
	var registeredCents int64
	for _, i := range members {
		a := acts[i]
		if a.PublishedAt.Before(c.First) {
			c.First = a.PublishedAt
		}
		if a.PublishedAt.After(c.Last) {
			c.Last = a.PublishedAt
		}
		if o := PrincipalOrgan(a.Organ); o != "" && !slices.Contains(c.Organs, o) {
			c.Organs = append(c.Organs, o)
		}
		switch PanelValueRoleOf(a.Type, a.Title, a.Head) {
		case PanelValueContracted:
			if a.ValueCents > c.ContractedCents {
				c.ContractedCents, c.LargestActID = a.ValueCents, a.ActID
			}
		case PanelValueRegistered:
			if a.ValueCents > registeredCents {
				registeredCents, registeredID = a.ValueCents, a.ActID
			}
		}
	}
	if c.ContractedCents == 0 {
		c.RegisteredCents, c.LargestActID = registeredCents, registeredID
	}
	sort.Strings(c.Organs)
	return c, c.ContractedCents > 0 || c.RegisteredCents > 0
}

func citesAnotherSupplier(others []string) bool {
	for _, o := range others {
		if _, public := PublicBody(o); !public {
			return true
		}
	}
	return false
}
