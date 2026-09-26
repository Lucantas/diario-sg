package domain

import (
	"encoding/csv"
	"strings"
	"testing"
	"time"
)

func csvLines(t *testing.T, text string) ([]string, []string) {
	t.Helper()
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = ';'
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows[0], rows[1]
}

func TestParseAmendmentReadsTheSaoGoncaloLine(t *testing.T) {
	header, row := csvLines(t, `"Código da Emenda";"Ano da Emenda";"Tipo de Emenda";"Código do Autor da Emenda";"Nome do Autor da Emenda";"Número da emenda";"Localidade de aplicação do recurso";"Código Município IBGE";"Município";"Código UF IBGE";"UF";"Região";"Código Função";"Nome Função";"Código Subfunção";"Nome Subfunção";"Código Programa";"Nome Programa";"Código Ação";"Nome Ação";"Código Plano Orçamentário";"Nome Plano Orçamentário";"Valor Empenhado";"Valor Liquidado";"Valor Pago";"Valor Restos A Pagar Inscritos";"Valor Restos A Pagar Cancelados";"Valor Restos A Pagar Pagos"
"202438050004";"2024";"Emenda Individual - Transferências com Finalidade Definida";"S/I";"S/I";"S/I";"SÃO GONÇALO - RJ";"3304904";"SÃO GONÇALO";"3300000";"RIO DE JANEIRO";"Sudeste";"10";"Saúde";"302";"Assistência hospitalar e ambulatorial";"5018";"ATENCAO ESPECIALIZADA";"2E90";"INCREMENTO TEMPORARIO";"0000";"X";"1000000,00";"500000,50";"-10,00";"0,00";"0,00";"0,00"`)
	cols, err := NewCSVColumns(header, AmendmentColumns...)
	if err != nil {
		t.Fatal(err)
	}

	a, err := ParseAmendment(cols, row)

	if err != nil || !IsSaoGoncaloAmendment(cols, row) || a.Year != 2024 || a.Author != "Sem informação" || a.CommittedCents != 100000000 ||
		a.LiquidatedCents != 50000050 || a.PaidCents != -1000 || a.Function != "Saúde" {
		t.Fatalf("veio %+v %v", a, err)
	}
}

func TestParseAmendmentPaymentKeepsOnlyCompaniesOfSaoGoncalo(t *testing.T) {
	header, row := csvLines(t, `"Código da Emenda";"Código do Autor da Emenda";"Nome do Autor da Emenda";"Número da emenda";"Tipo de Emenda";"Ano/Mês";"Código do Favorecido";"Favorecido";"Natureza Jurídica";"Tipo Favorecido";"UF Favorecido";"Município Favorecido";"Valor Recebido"
"202438050004";"3805";"PARLAMENTAR";"0004";"Emenda Individual";"202609";"37016159000104";"EMPRESA LTDA";"Sociedade Empresária Limitada";"Pessoa Jurídica";"RJ";"SÃO GONÇALO";"80017,25"`)
	cols, err := NewCSVColumns(header, AmendmentPaymentColumns...)
	if err != nil {
		t.Fatal(err)
	}

	p, err := ParseAmendmentPayment(cols, row)

	if err != nil || !IsSaoGoncaloCompanyPayment(cols, row) || p.CNPJ != "37016159000104" || p.ValueCents != 8001725 || p.Month.Format("01/2006") != "09/2026" {
		t.Fatalf("veio %+v %v", p, err)
	}
	person := append([]string(nil), row...)
	person[cols["Tipo Favorecido"]] = "Pessoa Fisica"
	if IsSaoGoncaloCompanyPayment(cols, person) {
		t.Fatal("pessoa física não entra")
	}
}

func TestParseTransferSkipsANonCNPJFavored(t *testing.T) {
	header, row := csvLines(t, `"ANO / MÊS";"TIPO TRANSFERÊNCIA";"TIPO FAVORECIDO";"UF";"CÓDIGO MUNICÍPIO SIAFI";"NOME MUNICÍPIO";"NOME ÓRGÃO";"NOME FUNÇÃO";"NOME PROGRAMA";"NOME AÇÃO";"LINGUAGEM CIDADÃ";"CÓDIGO FAVORECIDO";"NOME FAVORECIDO";"VALOR TRANSFERIDO"
"202601";"Legais, Voluntárias e Específicas";"Fundo Público";"RJ";"5897";"SAO GONCALO";"Ministério da Saúde";"Saúde";"ATENCAO";"PISO";"";"12345678000195";"FUNDO MUNICIPAL DE SAUDE DE SAO GONCALO";"1234567,89"`)
	cols, err := NewCSVColumns(header, TransferColumns...)
	if err != nil {
		t.Fatal(err)
	}

	tr, ok, err := ParseTransfer(cols, row)

	if err != nil || !ok || !IsSaoGoncaloTransfer(cols, row) || tr.ValueCents != 123456789 || tr.Organ != "Ministério da Saúde" {
		t.Fatalf("veio %+v %v %v", tr, ok, err)
	}
	masked := append([]string(nil), row...)
	masked[cols["CÓDIGO FAVORECIDO"]] = "***.123.456-**"
	if _, ok, _ := ParseTransfer(cols, masked); ok {
		t.Fatal("favorecido sem CNPJ não entra")
	}
}

func TestAggregateFavoredSumsByCNPJ(t *testing.T) {
	jan, feb := mustMonth("202601"), mustMonth("202602")
	got := AggregateFavored([]AmendmentPayment{
		{CNPJ: "1", Name: "A", Author: "Y", Month: feb, ValueCents: 10},
		{CNPJ: "2", Name: "B", Author: "X", Month: jan, ValueCents: 50},
		{CNPJ: "1", Name: "A", Author: "X", Month: jan, ValueCents: 60},
	})

	if len(got) != 2 || got[0].CNPJ != "1" || got[0].ValueCents != 70 || got[0].Payments != 2 || len(got[0].Authors) != 2 ||
		got[0].Authors[0] != "X" || !got[0].First.Equal(jan) || !got[0].Last.Equal(feb) {
		t.Fatalf("favorecidos: %+v", got)
	}
}

func mustMonth(s string) time.Time {
	m, err := time.Parse(federalMonthLayout, s)
	if err != nil {
		panic(err)
	}
	return m
}
