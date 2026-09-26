package domain

import (
	"strings"
	"testing"
)

const tceHeader = "\xef\xbb\xbfEnte;Unidade;Ano;Mes;NumeroEmpenho;TipoPessoa;CPFCNPJ;Funcao;Empenhado;Liquidado;Pago"

func tceColumns(t *testing.T) TCEColumns {
	t.Helper()
	c, err := NewTCEColumns(strings.Split(tceHeader, ";"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestParseTCECompanyRow(t *testing.T) {
	row := strings.Split("SAO GONCALO;FUNDO MUN SAUDE  SÃO GONÇALO;2025;06;123;JURÍDICA;39818737000151;SAÚDE;0.0;810250.72;-771358.685", ";")

	p, err := ParseTCERow(tceColumns(t), row)

	if err != nil {
		t.Fatal(err)
	}
	if p.Source != SourceTCE || p.Year != 2025 || p.Month != 6 || p.Unit != "FUNDO MUN SAUDE SÃO GONÇALO" || p.Commitment != "123" ||
		p.CNPJ != "39818737000151" || p.Function != "SAÚDE" || p.CommittedCents != 0 || p.LiquidatedCents != 81025072 || p.PaidCents != -77135869 {
		t.Errorf("empenho: %+v", p)
	}
}

func TestTCEPersonAndEmptyKindAreNotCompanies(t *testing.T) {
	c := tceColumns(t)
	for _, row := range []string{
		"SAO GONCALO;PREFEITURA;2025;1;1;FÍSICA;12345678901;ADMINISTRAÇÃO;1.0;1.0;1.0",
		"SAO GONCALO;PREFEITURA;2025;1;1;;00050000000000;ADMINISTRAÇÃO;1.0;1.0;1.0",
	} {
		if fields := strings.Split(row, ";"); c.IsCompany(fields) {
			t.Errorf("não é empresa: %s", row)
		} else if _, err := ParseTCERow(c, fields); err == nil {
			t.Errorf("deveria recusar: %s", row)
		}
	}
}

func TestTCEHeaderWithoutPaidIsRejected(t *testing.T) {
	if _, err := NewTCEColumns(strings.Split(strings.Replace(tceHeader, ";Pago", ";Valor", 1), ";")); err == nil {
		t.Error("cabeçalho sem Pago deveria dar erro")
	}
}

func TestTCEInvalidMonthIsAnError(t *testing.T) {
	row := strings.Split("SAO GONCALO;PREFEITURA;2025;13;1;JURÍDICA;39818737000151;SAÚDE;0.0;0.0;0.0", ";")

	if _, err := ParseTCERow(tceColumns(t), row); err == nil {
		t.Error("mês 13 deveria dar erro")
	}
}
