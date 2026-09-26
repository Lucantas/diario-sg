package domain

import (
	"sort"
	"strings"
	"time"
)

type StaffGroup struct {
	Group, Label string
}

type StaffGroupTotal struct {
	Headcount         int
	RemunerationCents int64
}

type StaffMonth struct {
	Month             time.Time
	Groups            []StaffGroupTotal
	Headcount         int
	RemunerationCents int64
	Appointments      int
	Dismissals        int
}

type StaffPanel struct {
	Units        []string
	Unit         string
	DiarioSource string
	Groups       []StaffGroup
	Months       []StaffMonth
}

var staffGroupOrder = []StaffGroup{
	{"Efetivo", "Efetivos"},
	{"Efetivo (com cargo ou função)", "Efetivos com cargo ou função"},
	{"Comissionado", "Comissionados"},
	{"Contratado", "Contratados"},
	{"Agente Político", "Agentes políticos"},
	{"Outros", "Outros"},
	{"Inativos", "Inativos e pensionistas"},
}

func StaffDiarioSource(unit string) string {
	if strings.Contains(strings.ToUpper(foldAccents(unit)), "CAMARA") {
		return SourceDiarioCamara
	}
	return SourceDiarioPrefeitura
}

func BuildStaffPanel(rows []StaffRow, counts []MonthlyActCount) []StaffMonth {
	groups := StaffGroupsOf(rows)
	index := make(map[string]int, len(groups))
	for i, g := range groups {
		index[g.Group] = i
	}
	byMonth := map[time.Time]*StaffMonth{}
	for _, r := range rows {
		m, ok := byMonth[r.Month]
		if !ok {
			m = &StaffMonth{Month: r.Month, Groups: make([]StaffGroupTotal, len(groups))}
			byMonth[r.Month] = m
		}
		g := &m.Groups[index[r.Group]]
		g.Headcount += r.Headcount
		g.RemunerationCents += r.RemunerationCents
		m.Headcount += r.Headcount
		m.RemunerationCents += r.RemunerationCents
	}
	for _, c := range counts {
		m, ok := byMonth[time.Date(c.Year, c.Month, 1, 0, 0, 0, 0, time.UTC)]
		if !ok {
			continue
		}
		switch c.Type {
		case ActNomeacao:
			m.Appointments += c.Count
		case ActExoneracao:
			m.Dismissals += c.Count
		}
	}
	out := make([]StaffMonth, 0, len(byMonth))
	for _, m := range byMonth {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Month.After(out[j].Month) })
	return out
}

func StaffGroupsOf(rows []StaffRow) []StaffGroup {
	present := map[string]bool{}
	for _, r := range rows {
		present[r.Group] = true
	}
	var out []StaffGroup
	for _, g := range staffGroupOrder {
		if present[g.Group] {
			out = append(out, g)
			delete(present, g.Group)
		}
	}
	var others []string
	for g := range present {
		others = append(others, g)
	}
	sort.Strings(others)
	for _, g := range others {
		out = append(out, StaffGroup{Group: g, Label: g})
	}
	return out
}
