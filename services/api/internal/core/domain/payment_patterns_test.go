package domain

import (
	"strings"
	"testing"
)

var coverage2021 = &PaymentCoverage{FromYear: 2021, FromMonth: 1, ToYear: 2026, ToMonth: 8}

func TestPaidWithoutPublicationSkipsCitedPublicAndSmallCreditors(t *testing.T) {
	profiles := map[string]SupplierProfile{
		"11111111000111": {CNPJ: "11111111000111", Name: "EMPRESA PAGA LTDA", LegalNature: "Sociedade Empresária Limitada"},
		"22222222000122": {CNPJ: "22222222000122", LegalNature: "Órgão Público do Poder Executivo Federal"},
	}
	paid := []CreditorPaid{
		{CNPJ: "11111111000111", PaidCents: 20_000_000, Years: []int{2022, 2024}},
		{CNPJ: "22222222000122", PaidCents: 90_000_000},
		{CNPJ: "33333333000133", PaidCents: 9_000_000},
		{CNPJ: "44444444000144", PaidCents: 50_000_000},
		{CNPJ: "28636579000100", PaidCents: 50_000_000},
	}

	got := FindPaidWithoutPublication(paid, map[string]bool{"44444444000144": true}, profiles)

	if len(got) != 1 || got[0].Profile.Name != "EMPRESA PAGA LTDA" {
		t.Fatalf("pago sem publicação: %+v", got)
	}
	if f := PaidWithoutPublicationFinding(got[0]); !strings.Contains(f.Title, "R$ 200.000,00 pagos de 2022 a 2024") {
		t.Errorf("título: %s", f.Title)
	}
}

func TestUnpaidContractLooksAtThePublicationYearAndTheNext(t *testing.T) {
	contracts := []SupplierContract{
		contract("11111111000111", panelDay(2022, 12, 28), 50_000_000, "sem pagamento"),
		contract("22222222000122", panelDay(2022, 12, 28), 50_000_000, "pago no ano seguinte"),
		contract("33333333000133", panelDay(2019, 5, 1), 50_000_000, "antes da cobertura"),
		contract("44444444000144", panelDay(2026, 3, 1), 50_000_000, "ano seguinte fora da cobertura"),
		contract("55555555000155", panelDay(2023, 3, 1), 5_000_000, "abaixo do piso"),
	}
	paid := []CreditorPaid{{CNPJ: "22222222000122", Years: []int{2023}, PaidCents: 1}}

	got := FindUnpaidContracts(contracts, paid, coverage2021, nil)

	if len(got) != 1 || got[0].Contract.ActID != "sem pagamento" {
		t.Fatalf("sem pagamento: %+v", got)
	}
	if FindUnpaidContracts(contracts, paid, nil, nil) != nil {
		t.Error("sem cobertura não há padrão")
	}
}

func TestPaidAboveAnnouncedNeedsTwiceAndOneMillionMore(t *testing.T) {
	contracts := []SupplierContract{
		contract("11111111000111", panelDay(2021, 1, 1), 50_000_000, "muito abaixo"),
		contract("22222222000122", panelDay(2021, 1, 1), 100_000_000, "pouco abaixo"),
		contract("33333333000133", panelDay(2021, 1, 1), 50_000_000, "aditivos cobrem"),
	}
	paid := []CreditorPaid{
		{CNPJ: "11111111000111", PaidCents: 200_000_000},
		{CNPJ: "22222222000122", PaidCents: 190_000_000},
		{CNPJ: "33333333000133", PaidCents: 200_000_000},
		{CNPJ: "44444444000144", PaidCents: 900_000_000},
	}

	got := FindPaidAboveAnnounced(contracts, map[string]int64{"33333333000133": 60_000_000}, paid, nil)

	if len(got) != 1 || got[0].Profile.CNPJ != "11111111000111" || got[0].AnnouncedCents != 50_000_000 {
		t.Fatalf("pago acima: %+v", got)
	}
}

func TestAttributeByNameUsesTheReadNameOfAUniqueCreditor(t *testing.T) {
	profiles := map[string]SupplierProfile{
		"11111111000111": {CNPJ: "11111111000111", Name: "FORCA AMBIENTAL LTDA."},
		"22222222000122": {CNPJ: "22222222000122", Name: "LIMPEZA TOTAL LTDA"},
		"33333333000133": {CNPJ: "33333333000133", Name: "LIMPEZA TOTAL LTDA"},
	}
	creditors := []CreditorPaid{{CNPJ: "11111111000111"}, {CNPJ: "22222222000122"}, {CNPJ: "33333333000133"}}
	acts := []PanelAct{
		{ActID: "forca", Head: "EXTRATO DO TERCEIRO TERMO ADITIVO. Partes: MUNICÍPIO DE SÃO GONÇALO e FORÇA AMBIENTAL LTDA, objeto: coleta"},
		{ActID: "ambigua", Head: "EXTRATO DE CONTRATO. Partes: MUNICÍPIO DE SÃO GONÇALO e LIMPEZA TOTAL LTDA, objeto: limpeza"},
		{ActID: "outra", Head: "EXTRATO DE CONTRATO. Partes: MUNICÍPIO DE SÃO GONÇALO e OUTRA EMPRESA LTDA, objeto: limpeza"},
	}

	got := AttributeByName(acts, creditors, profiles)

	if len(got) != 1 || got[0].ActID != "forca" || got[0].CNPJ != "11111111000111" {
		t.Fatalf("atribuídos: %+v", got)
	}
}
