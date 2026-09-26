package domain

import (
	"testing"
	"time"
)

func TestBuildStaffPanelSumsGroupsAndAddsTheDiarioCounts(t *testing.T) {
	jan, feb := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	rows := []StaffRow{
		{Month: jan, Unit: "PREFEITURA", Situation: "Efetivo - Estatutário", Group: "Efetivo", Headcount: 10, RemunerationCents: 1000},
		{Month: jan, Unit: "IPASG", Situation: "Efetivo - Celetista", Group: "Efetivo", Headcount: 5, RemunerationCents: 500},
		{Month: jan, Unit: "PREFEITURA", Situation: "Comissionado Extraquadro", Group: "Comissionado", Headcount: 3, RemunerationCents: 900},
		{Month: feb, Unit: "PREFEITURA", Situation: "Novo", Group: "Grupo novo", Headcount: 1, RemunerationCents: 1},
	}
	counts := []MonthlyActCount{{Type: ActNomeacao, Year: 2025, Month: time.January, Count: 7}, {Type: ActExoneracao, Year: 2025, Month: time.January, Count: 2},
		{Type: ActNomeacao, Year: 2023, Month: time.January, Count: 99}}

	months := BuildStaffPanel(rows, counts)
	groups := StaffGroupsOf(rows)

	if len(groups) != 3 || groups[0].Label != "Efetivos" || groups[1].Label != "Comissionados" || groups[2].Label != "Grupo novo" {
		t.Fatalf("grupos: %+v", groups)
	}
	if len(months) != 2 || !months[0].Month.Equal(feb) {
		t.Fatalf("meses: %+v", months)
	}
	j := months[1]
	if j.Headcount != 18 || j.RemunerationCents != 2400 || j.Groups[0].Headcount != 15 || j.Groups[1].RemunerationCents != 900 ||
		j.Appointments != 7 || j.Dismissals != 2 {
		t.Fatalf("janeiro: %+v", j)
	}
}

func TestStaffDiarioSourceFollowsTheUnit(t *testing.T) {
	if StaffDiarioSource("CÂMARA SÃO GONÇALO") != SourceDiarioCamara || StaffDiarioSource("") != SourceDiarioPrefeitura {
		t.Fatal("fonte do Diário errada")
	}
}

func TestBuildStaffPanelMatchesMonthsFromAnyLocation(t *testing.T) {
	fromDB := time.Date(2024, 1, 1, 0, 0, 0, 0, time.FixedZone("", 0))
	rows := []StaffRow{{Month: fromDB, Group: "Efetivo", Headcount: 1}, {Month: fromDB.In(time.UTC), Group: "Outros", Headcount: 1}}

	months := BuildStaffPanel(rows, []MonthlyActCount{{Type: ActNomeacao, Year: 2024, Month: time.January, Count: 115}})

	if len(months) != 1 || months[0].Appointments != 115 || months[0].Headcount != 2 {
		t.Fatalf("meses: %+v", months)
	}
}
