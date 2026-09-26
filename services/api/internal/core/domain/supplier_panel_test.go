package domain

import (
	"fmt"
	"testing"
	"time"
)

const (
	panelSupplier = "14180324000163"
	panelOther    = "32040529000125"
	homologHead   = "EXTRATO DA HOMOLOGAÇÃO DO PREGÃO ELETRÔNICO N°.\n90013/2025.\nNos termos do relatório final apresentado pelo Pregoeiro"
	contractHead  = "EXTRATO DO CONTRATO 010/SEMED/2025 DO PREGÃO\nProcesso: 7717/2025.\nPartes: MUNICÍPIO DE SÃO GONÇALO e F.P. VIEIRA ENGENHARIA"
	ataHead       = "EXTRATO DA ATA DE REGISTRO DE PREÇOS\nO MUNICÍPIO DE SÃO GONÇALO torna público"
)

func panelDay(y, m, d int) time.Time { return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC) }

func panelAct(id, cnpj, organ string, day time.Time, cents int64, head string, refs ...string) PanelAct {
	return PanelAct{ActID: id, CNPJ: cnpj, Organ: organ, PublishedAt: day, ValueCents: cents, Type: ActLicitacao, Title: head, Head: head, Refs: refs}
}

func TestSupplierPanelCountsEachContractOnceByItsContractValue(t *testing.T) {
	acts := []PanelAct{
		panelAct("h", panelSupplier, "SEMED", panelDay(2025, 5, 30), 10629252147, homologHead, "processo:77172025"),
		panelAct("c", panelSupplier, "SEMED", panelDay(2025, 5, 30), 10629252147, contractHead, "processo:77172025", "contrato:10/SEMED/2025"),
		panelAct("a", panelSupplier, "SEMED", panelDay(2025, 6, 2), 12000000000, ataHead, "processo:77172025"),
	}

	p := BuildSupplierPanel(acts, PanelFilter{})

	if p.Contracts != 1 || p.Suppliers != 1 || p.ContractedCents != 10629252147 || p.RegisteredCents != 0 {
		t.Fatalf("painel inesperado: %+v", p)
	}
	row := p.Rows[0]
	if row.CNPJ != panelSupplier || row.Contracts != 1 || row.LargestActID != "c" || !row.First.Equal(panelDay(2025, 5, 30)) || !row.Last.Equal(panelDay(2025, 6, 2)) {
		t.Fatalf("linha inesperada: %+v", row)
	}
}

func TestSupplierPanelKeepsPriceRegistryApart(t *testing.T) {
	acts := []PanelAct{
		panelAct("h", panelOther, "SEMMATRAN", panelDay(2026, 3, 12), 2849799998, "PREGÃO ELETRÔNICO Nº 90030/2025\nHomologo o resultado da licitação", "processo:26862025"),
		panelAct("a", panelOther, "SEMMATRAN", panelDay(2026, 3, 12), 2849799998, ataHead, "processo:26862025"),
	}

	p := BuildSupplierPanel(acts, PanelFilter{})

	if p.Contracts != 1 || p.ContractedCents != 0 || p.RegisteredCents != 2849799998 || p.Rows[0].LargestActID != "a" {
		t.Fatalf("ata deveria contar como registrado: %+v", p)
	}
}

func TestSupplierPanelJoinsActsWithoutReferencesBySameValueAndYear(t *testing.T) {
	acts := []PanelAct{
		panelAct("1", panelSupplier, "FMS", panelDay(2023, 7, 25), 1043928000, "PREGÃO ELETRÔNICO PARTES: FUNDAÇÃO MUNICIPAL DE SAÚDE"),
		panelAct("2", panelSupplier, "FMS", panelDay(2023, 9, 1), 1043928000, "PREGÃO ELETRÔNICO PARTES: FUNDAÇÃO MUNICIPAL DE SAÚDE"),
		panelAct("3", panelSupplier, "FMS", panelDay(2024, 7, 25), 1043928000, "PREGÃO ELETRÔNICO N° 016/2022 PARTES: FUNDAÇÃO MUNICIPAL DE SAÚDE"),
	}

	p := BuildSupplierPanel(acts, PanelFilter{})

	if p.Contracts != 2 || p.ContractedCents != 2*1043928000 {
		t.Fatalf("esperava uma contratação por ano: %+v", p)
	}
}

func TestSupplierPanelLeavesOutSharedActsAndPublicBodies(t *testing.T) {
	shared := panelAct("s", panelSupplier, "SEMED", panelDay(2025, 1, 1), 500000, contractHead, "processo:1")
	shared.OtherCNPJs = []string{panelOther, "28636579000100"}
	withPublicBody := panelAct("p", panelOther, "SEMED", panelDay(2025, 1, 1), 700000, contractHead, "processo:2")
	withPublicBody.OtherCNPJs = []string{"39260120000163"}
	acts := []PanelAct{
		shared,
		withPublicBody,
		panelAct("m", "28636579000100", "SEMED", panelDay(2025, 1, 1), 900000, contractHead, "processo:3"),
		panelAct("z", panelSupplier, "SEMED", panelDay(2025, 1, 1), 0, contractHead, "processo:4"),
	}

	p := BuildSupplierPanel(acts, PanelFilter{})

	if p.Suppliers != 1 || p.Rows[0].CNPJ != panelOther || p.ContractedCents != 700000 {
		t.Fatalf("só o ato de um fornecedor deveria contar: %+v", p)
	}
}

func TestSupplierPanelFiltersByYearAndOrganAndTotalsTheOtherAxis(t *testing.T) {
	acts := []PanelAct{
		panelAct("1", panelSupplier, "SEMED", panelDay(2024, 2, 1), 300000, contractHead, "processo:1"),
		panelAct("2", panelSupplier, "FMSSG", panelDay(2024, 3, 1), 200000, contractHead, "processo:2"),
		panelAct("3", panelOther, "FMS", panelDay(2025, 3, 1), 100000, contractHead, "processo:3"),
		panelAct("4", panelOther, "SEMED", panelDay(2025, 4, 1), 50000, ataHead, "processo:4"),
	}

	p := BuildSupplierPanel(acts, PanelFilter{Year: 2024, Organ: "FMS"})

	if p.Contracts != 1 || p.ContractedCents != 200000 || len(p.Rows) != 1 || p.Rows[0].Organs[0] != "FMS" {
		t.Fatalf("filtro de ano e órgão inesperado: %+v", p)
	}
	wantYears := []PanelTotal{{Key: "2024", Contracts: 1, ContractedCents: 200000}, {Key: "2025", Contracts: 1, ContractedCents: 100000}}
	if fmt.Sprint(p.Years) != fmt.Sprint(wantYears) {
		t.Fatalf("totais por ano deveriam respeitar o órgão: %+v", p.Years)
	}
	wantOrgans := []PanelTotal{{Key: "SEMED", Contracts: 1, ContractedCents: 300000}, {Key: "FMS", Contracts: 1, ContractedCents: 200000}}
	if fmt.Sprint(p.Organs) != fmt.Sprint(wantOrgans) {
		t.Fatalf("totais por órgão deveriam respeitar o ano: %+v", p.Organs)
	}
}

func TestSupplierPanelOrdersByContractedThenRegisteredAndLimitsRows(t *testing.T) {
	var acts []PanelAct
	for i := range SupplierPanelLimit + 5 {
		cnpj := fmt.Sprintf("%014d", 10000000000000+i)
		acts = append(acts, panelAct(fmt.Sprint(i), cnpj, "SEMED", panelDay(2025, 1, 1), int64(1000+i), contractHead, fmt.Sprintf("processo:%d", i)))
	}
	acts = append(acts, panelAct("ata", panelOther, "SEMED", panelDay(2025, 1, 1), 99999999, ataHead, "processo:x"))

	p := BuildSupplierPanel(acts, PanelFilter{})

	if p.Suppliers != SupplierPanelLimit+6 || len(p.Rows) != SupplierPanelLimit {
		t.Fatalf("esperava %d linhas de %d fornecedores: %d de %d", SupplierPanelLimit, SupplierPanelLimit+6, len(p.Rows), p.Suppliers)
	}
	if p.Rows[0].ContractedCents != int64(1000+SupplierPanelLimit+4) || p.Rows[1].ContractedCents >= p.Rows[0].ContractedCents {
		t.Fatalf("ordem inesperada: %+v", p.Rows[:2])
	}
	for _, r := range p.Rows {
		if r.CNPJ == panelOther {
			t.Fatalf("quem só tem ata vem depois de quem tem contrato: %+v", r)
		}
	}
}

const prorrogationHead = "PREGÃO ELETRÔNICO N° 016/2022 PARTES: FUNDAÇÃO MUNICIPAL DE SAÚDE E A4PM ANALYTICS LTDA. OBJETO: TERMO ADITIVO PARA PRORROGAR O PRAZO CONTRATUAL PELO PERÍODO DE 12 (DOZE) MESES. VALOR TOTAL DO CONTRATO: R$ 10.439.280,00"

func TestSupplierPanelCountsAmendmentsInTheYearTheyWerePublished(t *testing.T) {
	acts := []PanelAct{
		panelAct("c", panelSupplier, "FMS", panelDay(2022, 7, 25), 1043928000, contractHead, "contrato:16/2022"),
		panelAct("p23", panelSupplier, "FMS", panelDay(2023, 7, 20), 1043928000, prorrogationHead, "contrato:16/2022"),
		panelAct("p23r", panelSupplier, "FMS", panelDay(2023, 8, 1), 1043928000, prorrogationHead, "contrato:16/2022"),
		panelAct("p24", panelSupplier, "FMS", panelDay(2024, 7, 22), 1043928000, prorrogationHead, "contrato:16/2022"),
	}

	all := BuildSupplierPanel(acts, PanelFilter{})
	y2022 := BuildSupplierPanel(acts, PanelFilter{Year: 2022})
	y2023 := BuildSupplierPanel(acts, PanelFilter{Year: 2023})

	if all.Contracts != 1 || all.ContractedCents != 1043928000 || all.AmendedCents != 2087856000 || all.Rows[0].AmendedCents != 2087856000 {
		t.Fatalf("duas prorrogações, a republicada contando uma vez: %+v", all)
	}
	if y2022.AmendedCents != 0 || y2022.ContractedCents != 1043928000 {
		t.Fatalf("2022 só tem o contrato: %+v", y2022)
	}
	if y2023.Contracts != 0 || y2023.ContractedCents != 0 || y2023.AmendedCents != 1043928000 || y2023.Suppliers != 1 || y2023.Rows[0].Contracts != 0 {
		t.Fatalf("2023 só tem a prorrogação: %+v", y2023)
	}
	want := map[string]int64{"2022": 0, "2023": 1043928000, "2024": 1043928000}
	for _, y := range all.Years {
		if y.AmendedCents != want[y.Key] {
			t.Fatalf("aditivo por ano de publicação: %+v", all.Years)
		}
	}
}

func TestSupplierPanelAppliesDeclaredPercentToTheContractedValue(t *testing.T) {
	increase := panelAct("ad", panelSupplier, "FMS", panelDay(2016, 2, 10), 0, "EXTRATO DO PRIMEIRO TERMO ADITIVO AO CONTRATO FMS Nº 007/2015. OBJETO: acréscimo quantitativo equivalente a 20% do valor contratado.", "contrato:7/2015")
	increase.DeclaredIncreaseBP = 2000
	acts := []PanelAct{panelAct("c", panelSupplier, "FMS", panelDay(2015, 6, 1), 100000000, contractHead, "contrato:7/2015"), increase}

	p := BuildSupplierPanel(acts, PanelFilter{})

	if p.AmendedCents != 20000000 {
		t.Fatalf("20%% de R$ 1 milhão: %+v", p)
	}
}

func TestSupplierPanelKeepsContractingWithOnlyAmendmentsOutOfTheCount(t *testing.T) {
	acts := []PanelAct{panelAct("p", panelSupplier, "FMS", panelDay(2024, 7, 22), 1043928000, prorrogationHead, "contrato:16/2022")}

	p := BuildSupplierPanel(acts, PanelFilter{})

	if p.Contracts != 0 || p.Suppliers != 1 || p.AmendedCents != 1043928000 || p.Rows[0].LargestActID != "p" {
		t.Fatalf("contratação só com aditivo soma o aditivo e não conta: %+v", p)
	}
}
