package domain

import (
	"testing"
	"time"
)

const (
	semmaRatificacao = "RATIFICO a situação de dispensa de licitação: Processo nº: 21.577/2019. Valor Global: R$ 11.862,00. O presente Termo tem por fundamento legal o art. 24, inciso II c/c 23, II da Lei nº 8.666/93."
	semadRatificacao = "RATIFICO a situação de Dispensa de Licitação, baseada no art. 24, inciso II da Lei Federal nº. 8.666 de 21 de junho de 1993, Processo nº 53.639/2018, no valor de R$ 17.049,60."
	semadRepublicada = semadRatificacao + " Republicado por incorreção da PMSG."
	semtranLei14133  = "RATIFICO a DISPENSA DE LICITAÇÃO, conforme artigo 75, inciso II da Lei de Licitações n.º 14.133 de 1º de abril de 2021. VALOR TOTAL: R$ 57.999,60"
)

func dispensaAct(id, cnpj, organ string, day time.Time, cents int64, body string, processes ...string) DispensaAct {
	a := DispensaAct{ActID: id, CNPJ: cnpj, Organ: organ, PublishedAt: day, ValueCents: cents, Body: body}
	for _, p := range processes {
		a.Processes = append(a.Processes, DispensaProcess{Key: nonDigitRe.ReplaceAllString(p, ""), Label: p})
	}
	return a
}

func TestFindSplitDispensasFlagsTheSameSupplierAboveTheYearlyLimit(t *testing.T) {
	acts := []DispensaAct{
		dispensaAct("semma", "53775862000152", "SEMMA", civilDate(2020, 2, 19), 1186200, semmaRatificacao, "21.577/2019"),
		dispensaAct("semad", "53775862000152", "SEMAD", civilDate(2020, 5, 22), 1704960, semadRatificacao, "53.639/2018"),
		dispensaAct("semad-rep", "53775862000152", "SEMAD", civilDate(2020, 5, 28), 1704960, semadRepublicada, "56.639/2018"),
	}

	got := FindSplitDispensas(acts)

	if len(got) != 1 {
		t.Fatalf("esperava 1 caso, veio %+v", got)
	}
	s := got[0]
	if s.CNPJ != "53775862000152" || s.Year != 2020 || len(s.Contracts) != 2 || s.TotalCents != 2891160 || s.LimitCents != 1760000 {
		t.Fatalf("caso inesperado: %+v", s)
	}
	if ids := s.Contracts[1].ActIDs; len(ids) != 2 || ids[0] != "semad" || ids[1] != "semad-rep" {
		t.Errorf("a republicação deveria entrar na contratação da SEMAD: %+v", s.Contracts[1])
	}
}

func TestFindSplitDispensasCountsOneContractPerProcess(t *testing.T) {
	cnpj := "18657198000146"
	acts := []DispensaAct{
		dispensaAct("ratificacao", cnpj, "SEMTRAN", civilDate(2024, 10, 3), 5799960, semtranLei14133, "18.635/2024"),
		dispensaAct("termo", cnpj, "SEMTRAN", civilDate(2024, 10, 4), 5799960, semtranLei14133, "18.635/2024"),
		dispensaAct("contrato", cnpj, "SEMTRAN", civilDate(2024, 10, 9), 5799960, "Valor Global: R$ 57.999,60", "18.635/2024"),
	}

	if got := FindSplitDispensas(acts); len(got) != 0 {
		t.Fatalf("o mesmo processo publicado três vezes não é fracionamento: %+v", got)
	}
}

func TestFindSplitDispensasJoinsProcessesCitedByTheSameAct(t *testing.T) {
	cnpj := "63832662000148"
	body := "com fundamento no art. 75, inciso II, da Lei Federal nº 14.133/2021. Valor Total: R$ 53.000,00"
	acts := []DispensaAct{
		dispensaAct("edital-colado", cnpj, "SEMHAB", civilDate(2026, 7, 30), 5300000, body, "7405/2026", "07537/2026"),
		dispensaAct("contrato", cnpj, "SMTC", civilDate(2026, 9, 3), 5300000, body, "7405/2026"),
		dispensaAct("outro", cnpj, "SMTC", civilDate(2026, 9, 10), 5300000, body, "07537/2026"),
	}

	if got := FindSplitDispensas(acts); len(got) != 0 {
		t.Fatalf("processos citados no mesmo ato são uma contratação: %+v", got)
	}
}

func TestFindSplitDispensasJoinsActWithoutProcessBySameValue(t *testing.T) {
	cnpj := "11222333000181"
	body := "art. 75, inciso II, da Lei 14.133. Valor: R$ 40.000,00"
	acts := []DispensaAct{
		dispensaAct("a", cnpj, "SEMED", civilDate(2025, 2, 1), 4000000, body, "1.111/2025"),
		dispensaAct("a-sem-processo", cnpj, "SEMED", civilDate(2025, 2, 3), 4000000, body),
	}

	if got := FindSplitDispensas(acts); len(got) != 0 {
		t.Fatalf("ato sem processo com o mesmo valor é a mesma contratação: %+v", got)
	}
}

func TestFindSplitDispensasIgnoresWhatIsNotAValueDispensa(t *testing.T) {
	cnpj := "11222333000181"
	value := "art. 75, inciso II, da Lei 14.133."
	cases := map[string][]DispensaAct{
		"emergência": {
			dispensaAct("a", cnpj, "SEMED", civilDate(2025, 2, 1), 4000000, value, "1.111/2025"),
			dispensaAct("b", cnpj, "SEMED", civilDate(2025, 3, 1), 4000000, value+" contratação emergencial", "2.222/2025"),
		},
		"outro inciso": {
			dispensaAct("a", cnpj, "SEMED", civilDate(2025, 2, 1), 4000000, value, "1.111/2025"),
			dispensaAct("b", cnpj, "SEMED", civilDate(2025, 3, 1), 4000000, "art. 75, inciso VIII", "2.222/2025"),
		},
		"acima do limite": {
			dispensaAct("a", cnpj, "SEMED", civilDate(2025, 2, 1), 4000000, value, "1.111/2025"),
			dispensaAct("b", cnpj, "SEMED", civilDate(2025, 3, 1), 7000000, value, "2.222/2025"),
		},
		"órgão público": {
			dispensaAct("a", "39260120000163", "SEMED", civilDate(2025, 2, 1), 4000000, value, "1.111/2025"),
			dispensaAct("b", "39260120000163", "SEMED", civilDate(2025, 3, 1), 4000000, value, "2.222/2025"),
		},
		"anos diferentes": {
			dispensaAct("a", cnpj, "SEMED", civilDate(2024, 12, 20), 4000000, value, "1.111/2024"),
			dispensaAct("b", cnpj, "SEMED", civilDate(2025, 1, 10), 4000000, value, "2.222/2025"),
		},
	}
	for name, acts := range cases {
		if got := FindSplitDispensas(acts); len(got) != 0 {
			t.Errorf("%s: não deveria acionar, veio %+v", name, got)
		}
	}
}

func TestFindSplitDispensasKeepsEqualContractsApartWhenOneIsRepublished(t *testing.T) {
	cnpj := "11222333000181"
	body := "art. 24, inciso II da Lei 8.666. Valor: R$ 9.000,00"
	acts := []DispensaAct{
		dispensaAct("p1", cnpj, "SEMED", civilDate(2019, 3, 1), 900000, body, "1.111/2019"),
		dispensaAct("p2", cnpj, "SEMED", civilDate(2019, 4, 1), 900000, body, "2.222/2019"),
		dispensaAct("p2-rep", cnpj, "SEMED", civilDate(2019, 4, 8), 900000, body+" Republicado por incorreção.", "2.222/2019"),
	}

	got := FindSplitDispensas(acts)

	if len(got) != 1 || len(got[0].Contracts) != 2 || got[0].TotalCents != 1800000 {
		t.Fatalf("a republicação com processo fica na própria contratação: %+v", got)
	}
}

func TestFindSplitDispensasSkipsActsWithAnotherPrivateCNPJ(t *testing.T) {
	cnpj := "11222333000181"
	body := "art. 75, inciso II, da Lei 14.133."
	shared := dispensaAct("dois-fornecedores", cnpj, "SEMED", civilDate(2025, 3, 1), 4000000, body, "2.222/2025")
	shared.OtherCNPJs = []string{"99888777000166"}
	withMunicipality := dispensaAct("com-municipio", cnpj, "SEMED", civilDate(2025, 4, 1), 4000000, body, "3.333/2025")
	withMunicipality.OtherCNPJs = []string{"28636579000100"}
	acts := []DispensaAct{
		dispensaAct("a", cnpj, "SEMED", civilDate(2025, 2, 1), 4000000, body, "1.111/2025"),
		shared,
	}

	if got := FindSplitDispensas(acts); len(got) != 0 {
		t.Fatalf("o valor de um ato com dois fornecedores não é de nenhum deles: %+v", got)
	}
	if got := FindSplitDispensas(append(acts[:1:1], withMunicipality)); len(got) != 1 {
		t.Fatalf("o CNPJ do Município não conta como outro fornecedor: %+v", got)
	}
}

func TestFindSplitDispensasUsesTheLimitOfTheFirstPublication(t *testing.T) {
	cnpj := "11222333000181"
	body := "art. 24, inciso II da Lei 8.666."
	acts := []DispensaAct{
		dispensaAct("ratificacao", cnpj, "SEMED", civilDate(2018, 7, 10), 1000000, body, "1.111/2018"),
		dispensaAct("extrato", cnpj, "SEMED", civilDate(2018, 7, 20), 1000000, body, "1.111/2018"),
		dispensaAct("outra", cnpj, "SEMED", civilDate(2018, 8, 1), 500000, body, "2.222/2018"),
	}

	if got := FindSplitDispensas(acts); len(got) != 0 {
		t.Fatalf("R$ 10.000 em 10/07/2018 passa do limite de R$ 8.000 e não é dispensa por valor: %+v", got)
	}
}

func TestFindSplitDispensasFlagsWorksWithTheirOwnLimit(t *testing.T) {
	cnpj := "11222333000181"
	works := "RATIFICO a dispensa de licitação com fundamento no art. 24, inciso I, da Lei 8.666/93, para serviços de engenharia."
	acts := []DispensaAct{
		dispensaAct("obra-1", cnpj, "SEMDUR", civilDate(2019, 3, 1), 2500000, works, "1.000/2019"),
		dispensaAct("obra-2", cnpj, "SEMDUR", civilDate(2019, 8, 1), 2000000, works, "2.000/2019"),
		dispensaAct("compra", cnpj, "SEMDUR", civilDate(2019, 9, 1), 1500000, semmaRatificacao, "3.000/2019"),
	}

	got := FindSplitDispensas(acts)

	if len(got) != 1 || got[0].Category != DispensaWorks || got[0].LimitCents != 3300000 || got[0].TotalCents != 4500000 || len(got[0].Contracts) != 2 {
		t.Fatalf("esperava só o caso de obras, sem somar a compra: %+v", got)
	}
}

func TestFindSplitDispensasKeepsWorksBelowTheWorksLimit(t *testing.T) {
	cnpj := "11222333000181"
	works := "com fundamento no art. 75, inciso I, da Lei 14.133/2021, obra de reforma."
	acts := []DispensaAct{
		dispensaAct("obra-1", cnpj, "SEMDUR", civilDate(2024, 3, 1), 6000000, works, "1.000/2024"),
		dispensaAct("obra-2", cnpj, "SEMDUR", civilDate(2024, 8, 1), 5000000, works, "2.000/2024"),
	}

	if got := FindSplitDispensas(acts); len(got) != 0 {
		t.Fatalf("R$ 110 mil em obras está abaixo do limite de 2024: %+v", got)
	}
}

func TestFindSplitDispensasCountsActCitingBothIncisosAsGoods(t *testing.T) {
	cnpj := "11222333000181"
	both := "FUNDAMENTO LEGAL: ARTIGO 24, INCISOS I E II DA LEI FEDERAL N.º 8.666/93."
	acts := []DispensaAct{
		dispensaAct("a", cnpj, "SEMDUR", civilDate(2020, 3, 1), 1000000, both, "1.000/2020"),
		dispensaAct("b", cnpj, "SEMDUR", civilDate(2020, 8, 1), 1000000, both, "2.000/2020"),
	}

	got := FindSplitDispensas(acts)

	if len(got) != 1 || got[0].Category != DispensaGoods || got[0].LimitCents != 1760000 {
		t.Fatalf("incisos I e II juntos ficam no limite de compras: %+v", got)
	}
}
