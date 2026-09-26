package domain

import (
	"sort"
	"strconv"
)

func WithPaid(p SupplierPanel, source string, paid []PaidTotal) SupplierPanel {
	byCNPJ := map[string]int64{}
	byYear := map[int]int64{}
	for _, t := range paid {
		if _, public := PublicBody(t.CNPJ); public || !t.paidBy(source) {
			continue
		}
		byYear[t.Year] += t.PaidCents
		if p.Filter.Year == 0 || t.Year == p.Filter.Year {
			byCNPJ[t.CNPJ] += t.PaidCents
			p.PaidCents += t.PaidCents
		}
	}
	rows := make([]SupplierRow, len(p.Rows))
	for i, row := range p.Rows {
		row.PaidCents = byCNPJ[row.CNPJ]
		rows[i] = row
	}
	p.Rows = rows
	p.Years = yearsWithPaid(p.Years, byYear)
	return p
}

func yearsWithPaid(years []PanelTotal, paid map[int]int64) []PanelTotal {
	out := make([]PanelTotal, 0, len(years)+len(paid))
	seen := map[string]bool{}
	for _, y := range years {
		year, _ := strconv.Atoi(y.Key)
		y.PaidCents = paid[year]
		seen[y.Key] = true
		out = append(out, y)
	}
	for year, cents := range paid {
		if key := strconv.Itoa(year); !seen[key] {
			out = append(out, PanelTotal{Key: key, PaidCents: cents})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
