package domain

import (
	"strings"
	"testing"
	"time"
)

var testCodes = RegistryCodes{
	Activities: map[string]string{"4120400": "Construção de edifícios", "4399103": "Obras de alvenaria", "7112000": "Serviços de engenharia"},
	Cities:     map[string]string{"5869": "SAO GONCALO"},
	Natures:    map[string]string{"2062": "Sociedade Empresária Limitada"},
	Roles:      map[string]string{"49": "Sócio-Administrador", "22": "Sócio"},
	Reasons:    map[string]string{"00": "Sem motivo", "01": "Extinção por encerramento liquidação voluntária"},
}

func fields(line string) []string {
	return strings.Split(strings.ReplaceAll(line, `"`, ""), ";")
}

func TestParseCompanyRow(t *testing.T) {
	c, err := ParseCompanyRow(fields(`"07396865";"CONSTRUTORA EXEMPLO LTDA";"2062";"49";"1500000,00";"03";""`), testCodes)
	if err != nil {
		t.Fatal(err)
	}
	want := RegistryCompany{Base: "07396865", Name: "CONSTRUTORA EXEMPLO LTDA", LegalNature: "Sociedade Empresária Limitada", CapitalCents: 150000000, Size: "Empresa de pequeno porte"}
	if c != want {
		t.Errorf("esperava %+v, veio %+v", want, c)
	}
}

func TestCompanySizes(t *testing.T) {
	for code, want := range map[string]string{"00": "Não informado", "01": "Microempresa", "03": "Empresa de pequeno porte", "05": "Demais"} {
		c, err := ParseCompanyRow(fields(`"1";"X";"2062";"49";"0,00";"`+code+`";""`), testCodes)
		if err != nil || c.Size != want {
			t.Errorf("porte %s: esperava %q, veio %q (%v)", code, want, c.Size, err)
		}
	}
}

func TestParseEstablishmentRow(t *testing.T) {
	row := `"07396865";"0001";"04";"1";"EXEMPLO OBRAS";"02";"20050518";"00";"";"";"20050518";"4120400";"4399103,7112000";"RUA";"DOUTOR NILO PECANHA";"120";"SALA  2      ";"CENTRO";"24445300";"RJ";"5869";"21";"26010000";"";"";"";"";"CONTATO@EXEMPLO.COM.BR";"";""`
	e, err := ParseEstablishmentRow(fields(row), testCodes)
	if err != nil {
		t.Fatal(err)
	}
	since := time.Date(2005, 5, 18, 0, 0, 0, 0, time.UTC)
	if e.CNPJ != "07396865000104" || !e.Headquarters || e.TradeName != "EXEMPLO OBRAS" || e.Status != "Ativa" ||
		!e.StatusSince.Equal(since) || !e.OpenedAt.Equal(since) || e.StatusReason != "Sem motivo" {
		t.Errorf("cabeçalho do estabelecimento: %+v", e)
	}
	if e.MainActivity != (Activity{"4120400", "Construção de edifícios"}) || len(e.OtherActivities) != 2 || e.OtherActivities[1].Description != "Serviços de engenharia" {
		t.Errorf("atividades: %+v %+v", e.MainActivity, e.OtherActivities)
	}
	if e.Street != "RUA DOUTOR NILO PECANHA" || e.Number != "120" || e.Complement != "SALA 2" || e.District != "CENTRO" ||
		e.ZIP != "24445300" || e.City != "SAO GONCALO" || e.UF != "RJ" {
		t.Errorf("endereço: %+v", e)
	}
}

func TestEstablishmentStatusesAndEmptyDates(t *testing.T) {
	for code, want := range map[string]string{"01": "Nula", "02": "Ativa", "03": "Suspensa", "04": "Inapta", "08": "Baixada"} {
		row := `"1";"0002";"00";"2";"";"` + code + `";"0";"01";"";"";"00000000";"4120400";"";"";"";"";"";"";"";"RJ";"5869";"";"";"";"";"";"";"";"";""`
		e, err := ParseEstablishmentRow(fields(row), testCodes)
		if err != nil {
			t.Fatal(err)
		}
		if e.Status != want || e.Headquarters || e.StatusSince != nil || e.OpenedAt != nil || len(e.OtherActivities) != 0 {
			t.Errorf("situação %s: %+v", code, e)
		}
	}
}

func TestParsePartnerRow(t *testing.T) {
	p, err := ParsePartnerRow(fields(`"07396865";"2";"FULANO DE TAL";"***123456**";"49";"20050518";"";"***000000**";"";"00";"6"`), testCodes)
	if err != nil {
		t.Fatal(err)
	}
	since := time.Date(2005, 5, 18, 0, 0, 0, 0, time.UTC)
	if p.Base != "07396865" || p.Kind != PartnerPerson || p.Name != "FULANO DE TAL" || p.Document != "***123456**" ||
		p.Role != "Sócio-Administrador" || !p.Since.Equal(since) {
		t.Errorf("sócio: %+v", p)
	}
}

func TestRowsWithWrongFieldCountAreRejected(t *testing.T) {
	if _, err := ParseCompanyRow([]string{"1", "X"}, testCodes); err == nil {
		t.Error("empresa com campos faltando deveria dar erro")
	}
	if _, err := ParseEstablishmentRow([]string{"1"}, testCodes); err == nil {
		t.Error("estabelecimento com campos faltando deveria dar erro")
	}
	if _, err := ParsePartnerRow([]string{"1"}, testCodes); err == nil {
		t.Error("sócio com campos faltando deveria dar erro")
	}
}

func TestUnknownCodesKeepTheCode(t *testing.T) {
	c, err := ParseCompanyRow(fields(`"1";"X";"9999";"49";"10,5";"05";""`), testCodes)
	if err != nil {
		t.Fatal(err)
	}
	if c.LegalNature != "9999" || c.CapitalCents != 1050 {
		t.Errorf("código desconhecido e capital: %+v", c)
	}
}

func TestCNPJBase(t *testing.T) {
	if got := CNPJBase("07396865000104"); got != "07396865" {
		t.Errorf("veio %q", got)
	}
}
