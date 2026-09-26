package domain

import (
	"strings"
	"testing"
	"time"
)

func contract(cnpj string, day time.Time, cents int64, id string) SupplierContract {
	return SupplierContract{CNPJ: cnpj, First: day, ContractedCents: cents, ActID: id}
}

func opened(day time.Time) *time.Time { return &day }

func TestSupplierContractsKeepsOnlyContractsWithValue(t *testing.T) {
	acts := []PanelAct{
		panelAct("c", panelSupplier, "SEMED", panelDay(2025, 5, 30), 10000000, contractHead, "contrato:10/SEMED/2025"),
		panelAct("c2", panelSupplier, "SEMED", panelDay(2025, 6, 2), 10000000, contractHead, "contrato:10/SEMED/2025"),
	}

	got := SupplierContracts(acts)

	if len(got) != 1 || got[0].ActID != "c" || got[0].ContractedCents != 10000000 || !got[0].First.Equal(panelDay(2025, 5, 30)) {
		t.Fatalf("contratações: %+v", got)
	}
}

func TestNewCompanyIsTheFirstContractSoonAfterTheHeadquartersOpened(t *testing.T) {
	profiles := map[string]SupplierProfile{
		"11111111000111": {CNPJ: "11111111000111", Headquarters: true, OpenedAt: opened(panelDay(2025, 1, 1))},
		"22222222000122": {CNPJ: "22222222000122", Headquarters: true, OpenedAt: opened(panelDay(2025, 1, 1))},
		"33333333000233": {CNPJ: "33333333000233", Headquarters: false, OpenedAt: opened(panelDay(2025, 1, 1))},
		"44444444000144": {CNPJ: "44444444000144", Headquarters: true, OpenedAt: opened(panelDay(2025, 6, 1))},
	}
	contracts := []SupplierContract{
		contract("11111111000111", panelDay(2025, 3, 1), 100, "novo"),
		contract("11111111000111", panelDay(2025, 4, 1), 100, "segundo"),
		contract("22222222000122", panelDay(2025, 12, 1), 100, "antigo"),
		contract("33333333000233", panelDay(2025, 2, 1), 100, "filial"),
		contract("44444444000144", panelDay(2025, 3, 1), 100, "antes"),
	}

	got := FindNewCompanyContracts(contracts, profiles)

	if len(got) != 1 || got[0].Contract.ActID != "novo" || got[0].Days != 59 {
		t.Fatalf("empresas novas: %+v", got)
	}
}

func TestUndercapitalizedNeedsShareCapitalAndTheMinimumValue(t *testing.T) {
	profiles := map[string]SupplierProfile{
		"11111111000111": {CNPJ: "11111111000111", CapitalCents: 1_000_000, LegalNature: "Sociedade Empresária Limitada"},
		"22222222000122": {CNPJ: "22222222000122", CapitalCents: 100_000, LegalNature: "Cooperativa"},
		"33333333000133": {CNPJ: "33333333000133", CapitalCents: 100, LegalNature: "Empresário (Individual)"},
		"44444444000144": {CNPJ: "44444444000144", CapitalCents: 5_000_000, LegalNature: "Sociedade Empresária Limitada"},
	}
	contracts := []SupplierContract{
		contract("11111111000111", panelDay(2025, 1, 1), 20_000_000, "pequena"),
		contract("11111111000111", panelDay(2025, 2, 1), 50_000_000, "maior"),
		contract("22222222000122", panelDay(2025, 1, 1), 50_000_000, "cooperativa"),
		contract("33333333000133", panelDay(2025, 1, 1), 29_000, "abaixo do piso"),
		contract("44444444000144", panelDay(2025, 1, 1), 50_000_000, "dez por cento"),
	}

	got := FindUndercapitalizedContracts(contracts, profiles)

	if len(got) != 1 || got[0].Contract.ActID != "maior" {
		t.Fatalf("capital: %+v", got)
	}
}

func TestSharedPartnersGroupDifferentCompaniesOnce(t *testing.T) {
	person := PartnerKey{Kind: PartnerPerson, Name: "FULANO DE TAL", Document: "***123456**"}
	holding := PartnerKey{Kind: PartnerCompany, Name: "HOLDING X LTDA", Document: "99888777000166"}
	homonym := PartnerKey{Kind: PartnerPerson, Name: "FULANO DE TAL", Document: "***999999**"}
	profiles := map[string]SupplierProfile{
		"11111111000111": {CNPJ: "11111111000111", Partners: []PartnerKey{person, holding}},
		"11111111000292": {CNPJ: "11111111000292", Partners: []PartnerKey{person, holding}},
		"22222222000122": {CNPJ: "22222222000122", Partners: []PartnerKey{holding, {Kind: PartnerPerson, Name: "Fulano de Tal", Document: "***123456**"}}},
		"33333333000133": {CNPJ: "33333333000133", Partners: []PartnerKey{homonym}},
	}
	contracts := []SupplierContract{
		contract("11111111000111", panelDay(2025, 1, 1), 100, "a"),
		contract("11111111000292", panelDay(2025, 2, 1), 100, "a2"),
		contract("22222222000122", panelDay(2025, 1, 1), 100, "b"),
		contract("33333333000133", panelDay(2025, 1, 1), 100, "c"),
	}

	got := FindSharedPartners(contracts, profiles)

	if len(got) != 1 || len(got[0].Suppliers) != 2 || len(got[0].Partners) != 2 {
		t.Fatalf("grupos: %+v", got)
	}
	f := SharedPartnerFinding(got[0])
	if strings.Contains(f.Title+f.Detail, "FULANO") || !strings.Contains(f.Title, "HOLDING X LTDA") || !strings.Contains(f.Title, "1 sócio pessoa física") ||
		!strings.HasPrefix(f.Title, "11.111.111/0001-11 e 22.222.222/0001-22: ") {
		t.Errorf("achado não pode nomear a pessoa física: %+v", f)
	}
}

func TestSharedAddressNeedsTheSameFullAddressWithNumber(t *testing.T) {
	profiles := map[string]SupplierProfile{
		"11111111000111": {CNPJ: "11111111000111", Address: "RUA JOSÉ FIGUEIREDO, 38, SALA 101, CENTRO, NITERÓI, RJ, 24030055"},
		"22222222000122": {CNPJ: "22222222000122", Address: "Rua Jose Figueiredo 38 sala 101, Centro, Niteroi, RJ, 24030055"},
		"33333333000133": {CNPJ: "33333333000133", Address: "RUA JOSE FIGUEIREDO, 38, SALA 102, CENTRO, NITEROI, RJ, 24030055"},
		"44444444000144": {CNPJ: "44444444000144", Address: "ESTRADA SEM NUMERO, ZONA RURAL"},
		"55555555000155": {CNPJ: "55555555000155", Address: "ESTRADA SEM NUMERO, ZONA RURAL"},
	}
	var contracts []SupplierContract
	for cnpj := range profiles {
		contracts = append(contracts, contract(cnpj, panelDay(2025, 1, 1), 100, cnpj))
	}

	got := FindSharedAddresses(contracts, profiles)

	if len(got) != 1 || len(got[0].Suppliers) != 2 || got[0].Suppliers[0].CNPJ != "11111111000111" {
		t.Fatalf("endereços: %+v", got)
	}
}

func TestSanctionedContractNeedsASanctionThatReachesTheCityOnTheDay(t *testing.T) {
	start, end := panelDay(2025, 1, 1), panelDay(2025, 12, 31)
	sanction := func(cnpj, category, scope, organ string) Sanction {
		return Sanction{Register: RegisterCEIS, Code: category, CNPJ: cnpj, Category: category, Scope: scope, Organ: organ, StartsAt: &start, EndsAt: &end}
	}
	sanctions := []Sanction{
		sanction("11111111000292", "Declaração de Inidoneidade com prazo determinado", "Em todos os Poderes da Esfera", "TCU"),
		sanction("22222222000122", "Impedimento/proibição de contratar com prazo determinado", "Em todos os Poderes da Esfera", "Prefeitura de Joinville (SC)"),
		sanction("33333333000133", "Suspensão", "No órgão sancionador", "CÂMARA MUNICIPAL DE SÃO GONÇALO - RJ"),
		sanction("44444444000144", "Multa", "Todas as Esferas em todos os Poderes", "CGU"),
	}
	contracts := []SupplierContract{
		contract("11111111000111", panelDay(2025, 6, 1), 100, "inidonea"),
		contract("11111111000111", panelDay(2026, 6, 1), 100, "depois"),
		contract("22222222000122", panelDay(2025, 6, 1), 100, "outro ente"),
		contract("33333333000133", panelDay(2025, 6, 1), 100, "sao goncalo"),
		contract("44444444000144", panelDay(2025, 6, 1), 100, "todas as esferas"),
	}

	got := FindSanctionedContracts(contracts, map[string]SupplierProfile{}, sanctions)

	var ids []string
	for _, s := range got {
		ids = append(ids, s.Contract.ActID)
	}
	if strings.Join(ids, ",") != "inidonea,sao goncalo,todas as esferas" {
		t.Fatalf("sancionadas: %v", ids)
	}
}
