package domain

import (
	"strings"
	"testing"
	"time"
)

func license(cnpj string, day time.Time, id, title, head string) LicenseAct {
	return LicenseAct{ActID: id, CNPJ: cnpj, PublishedAt: day, Title: title, Head: head}
}

func TestNewCompanyLicenseIsAGrantWithinTwoYearsOfTheHeadquartersOpening(t *testing.T) {
	profiles := map[string]SupplierProfile{
		"58452347000155": {CNPJ: "58452347000155", Name: "MICAL INVEST LTDA", Headquarters: true, OpenedAt: opened(panelDay(2024, 12, 13))},
		"22222222000122": {CNPJ: "22222222000122", Headquarters: true, OpenedAt: opened(panelDay(2020, 1, 1))},
		"33333333000233": {CNPJ: "33333333000233", Headquarters: false, OpenedAt: opened(panelDay(2026, 1, 1))},
		"44444444000144": {CNPJ: "44444444000144", Headquarters: true, OpenedAt: opened(panelDay(2026, 1, 1))},
	}
	grant := "CONCESSÃO DE LICENÇA"
	licenses := []LicenseAct{
		license("58452347000155", panelDay(2026, 6, 12), "mical", grant, "Licença Municipal Prévia LMP nº 008/2026"),
		license("58452347000155", panelDay(2026, 8, 1), "mical-li", grant, "Licença Municipal de Instalação"),
		license("22222222000122", panelDay(2026, 6, 12), "antiga", grant, ""),
		license("33333333000233", panelDay(2026, 6, 12), "filial", grant, ""),
		license("44444444000144", panelDay(2026, 3, 1), "pedido", "EMPRESA X LTDA", "EMPRESA X LTDA torna público que requereu à SEMMATRAN a Licença Municipal Prévia"),
		license("44444444000144", panelDay(2026, 4, 1), "cancelada", "CANCELAMENTO DE LICENÇA", ""),
		license("55555555000155", panelDay(2026, 4, 1), "sem-cadastro", grant, ""),
	}

	got := FindNewCompanyLicenses(licenses, profiles)

	if len(got) != 1 {
		t.Fatalf("esperava só a Mical: %+v", got)
	}
	if got[0].Days != 546 || got[0].License.ActID != "mical" || strings.Join(got[0].ActIDs, ",") != "mical,mical-li" {
		t.Errorf("achado inesperado: %+v", got[0])
	}
}

func TestNewCompanyLicenseLimitIsTwoYears(t *testing.T) {
	profiles := map[string]SupplierProfile{"11111111000111": {CNPJ: "11111111000111", Headquarters: true, OpenedAt: opened(panelDay(2024, 1, 1))}}
	opening := panelDay(2024, 1, 1)
	inside := license("11111111000111", opening.AddDate(0, 0, NewCompanyLicenseDays-1), "dentro", "CONCESSÃO DE LICENÇA", "")
	outside := license("11111111000111", opening.AddDate(0, 0, NewCompanyLicenseDays), "fora", "CONCESSÃO DE LICENÇA", "")

	if got := FindNewCompanyLicenses([]LicenseAct{outside}, profiles); len(got) != 0 {
		t.Errorf("730 dias não entra: %+v", got)
	}
	if got := FindNewCompanyLicenses([]LicenseAct{inside}, profiles); len(got) != 1 {
		t.Errorf("729 dias entra: %+v", got)
	}
}

func TestNewCompanyLicenseFinding(t *testing.T) {
	n := NewCompanyLicense{
		Profile: SupplierProfile{CNPJ: "58452347000155", Name: "MICAL INVEST LTDA", OpenedAt: opened(panelDay(2024, 12, 13))},
		License: license("58452347000155", panelDay(2026, 6, 12), "mical", "CONCESSÃO DE LICENÇA", ""),
		Days:    546, ActIDs: []string{"mical", "mical-li"},
	}

	f := NewCompanyLicenseFinding(n)

	if f.Title != "MICAL INVEST LTDA (58.452.347/0001-55): licença ambiental 546 dias depois da abertura" {
		t.Errorf("título: %q", f.Title)
	}
	if f.Detail != "Aberta em 13/12/2024; primeira licença publicada em 12/06/2026. 2 licenças da empresa nos dois anos depois da abertura." {
		t.Errorf("detalhe: %q", f.Detail)
	}
	if len(f.ActIDs) != 2 || !f.Mentions(EntityCNPJ, "58452347000155") {
		t.Errorf("achado: %+v", f)
	}
	if _, ok := PatternCatalog()[PatternNewCompanyLicense]; !ok {
		t.Error("padrão fora do catálogo")
	}
}
