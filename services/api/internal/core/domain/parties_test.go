package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestPartiesOfNamesEachCNPJ(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []Party
	}{
		{"partes com o Município e uma empresa",
			"Partes: MUNICÍPIO DE SÃO GONÇALO, CNPJ 28.636.579/0001-00 e CAIXA ECONÔMICA FEDERAL, CNPJ 00.360.305/0001-04.",
			[]Party{{CNPJ: "28636579000100", Name: "MUNICÍPIO DE SÃO GONÇALO", PublicBody: "Município de São Gonçalo"},
				{CNPJ: "00360305000104", Name: "CAIXA ECONÔMICA FEDERAL"}}},
		{"empresa inscrita no C.N.P.J. com quebra de linha",
			"Empresa: ALL FOOD SERVIÇOS E COMÉRCIO DE\nALIMENTOS LTDA-ME, inscrita no C.N.P.J. sob o nº 01.742.126/0001-02, com sede",
			[]Party{{CNPJ: "01742126000102", Name: "ALL FOOD SERVIÇOS E COMÉRCIO DE ALIMENTOS LTDA-ME"}}},
		{"secretaria e empresa com CNPJ nº:",
			"PARTES: SECRETARIA MUNICIPAL DE TRANSPORTES e LM\nCURSOS DE TRANSITO SOCIEDADE UNIPESSOAL LTDA - CNPJ nº:\n18.657.198/0001-46.",
			[]Party{{CNPJ: "18657198000146", Name: "LM CURSOS DE TRANSITO SOCIEDADE UNIPESSOAL LTDA"}}},
		{"a empresa depois de outra parte com nome em caixa mista",
			"entre a Fundação de Artes, Esporte e Lazer de São Gonçalo e a empresa NP TECNOLOGIA E GESTÃO DE DADOS LTDA - CNPJ. 07.797.967/0001-95,",
			[]Party{{CNPJ: "07797967000195", Name: "NP TECNOLOGIA E GESTÃO DE DADOS LTDA"}}},
		{"CNPJ de outra parte logo antes",
			"Partes: MUNICÍPIO, CNPJ nº 28.636.579/0001-00, CENTRO DE INTEGRAÇÃO EMPRESA ESCOLA – CIEE RJ, CNPJ nº 33.661.745/0001-50.",
			[]Party{{CNPJ: "28636579000100", Name: "MUNICÍPIO", PublicBody: "Município de São Gonçalo"},
				{CNPJ: "33661745000150", Name: "CENTRO DE INTEGRAÇÃO EMPRESA ESCOLA – CIEE RJ"}}},
		{"endereço entre o nome e o CNPJ",
			"Empresa: PRIOM TECNOLOGIA EIRELI, estabelecida na Rua Taquaruçú, nº 465, inscrita no CNPJ sob o nº 11.619.992/0001-56.",
			[]Party{{CNPJ: "11619992000156", Name: "PRIOM TECNOLOGIA EIRELI"}}},
		{"o mesmo CNPJ duas vezes vira uma parte",
			"Contratada: INVICTTA LTDA, CNPJ 10.746.140/0001-67. Pagamento à INVICTTA LTDA, CNPJ 10.746.140/0001-67.",
			[]Party{{CNPJ: "10746140000167", Name: "INVICTTA LTDA"}}},
	}
	for _, c := range cases {
		if got := PartiesOf(c.body); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n veio     %+v\n esperava %+v", c.name, got, c.want)
		}
	}
}

func TestPartyNameDoesNotStartMidWord(t *testing.T) {
	body := strings.Repeat("x", 190) + " Presidente da Fundação DIMASTER LTDA, CNPJ 02.520.829/0001-40."

	got := PartiesOf(body)

	if len(got) != 1 || strings.HasPrefix(got[0].Name, "x") || !strings.HasSuffix(got[0].Name, "DIMASTER LTDA") {
		t.Fatalf("nome cortado no meio de uma palavra: %+v", got)
	}
}

func TestParliamentaryQuotaOf(t *testing.T) {
	body := `TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS
DE COTA PARA O EXERCÍCIO DA ATIVIDADE
PARLAMENTAR MUNICIPAL- CEAPM OUTUBRO 2025
JUAN OLIVEIRA
Processo n: 1198/2025
Tendo em vista o parecer da Comissão de Prestação de Contas
de Cota para Exercício da Atividade Parlamentar, constante as
folhas 80 a 83, autorizo a publicação da prestação de contas de
Cota para o Exercício da Atividade Parlamentar Municipal -
CEAPM que foi APROVADA, apresentada pelo a JUAN PATRICK
PINHEIRO DE OLIVEIRA – Vereador JUAN OLIVEIRA, relativo
ao mês de OUTUBRO de 2025, no valor de R$ 10.000,00 (dez
mil reais).`

	got, ok := ParliamentaryQuotaOf(ActPrestacaoContas, body)

	want := ParliamentaryQuota{Councillor: "JUAN OLIVEIRA", FullName: "JUAN PATRICK PINHEIRO DE OLIVEIRA", Month: "2025-10", ValueCents: 1000000}
	if !ok || got != want {
		t.Fatalf("veio %+v %v", got, ok)
	}
	noComma := "CEAPM que foi APROVADA, apresentada pelo a ALÉCIO\nBREDA DIAS – VEREADOR LECINHO relativo ao mês de\nOUTUBRO de 2025, no valor de R$ 10.000,00 (dez mil reais)."
	if q, ok := ParliamentaryQuotaOf(ActPrestacaoContas, noComma); !ok || q.Councillor != "LECINHO" || q.FullName != "ALÉCIO BREDA DIAS" {
		t.Fatalf("sem vírgula antes de relativo: %+v %v", q, ok)
	}
	glued := "CEAPM que foi APROVADA, apresentada pelo a Vereador\nRODRIGO DUARTE BASTOS– Vereador RODRIGO DUARTE,\nrelativo ao mês de NOVEMBRO de 2025, no valor de R$\n10.000,00 (dez mil reais)."
	if q, ok := ParliamentaryQuotaOf(ActPrestacaoContas, glued); !ok || q.FullName != "RODRIGO DUARTE BASTOS" || q.Month != "2025-11" {
		t.Fatalf("travessão colado e Vereador antes do nome civil: %+v %v", q, ok)
	}
	for text, want := range map[string]string{
		"apresentada Vereador Sebastião Victor Gonçalves Pereira –\nVereador Tião – relativo ao mês de FEVEREIRO de 2026, no\nvalor de R$ 13.000,00": "Tião",
		"apresentada por Alecio Breda Dias –\nVereador Lecinho, referente ao mês de JUNHO de 2026, no valor\nde R$ 13.000,00":                        "Lecinho",
		"apresentada pelo ISAAC SOUZA\nDA SILVA RICALDE – Vereador ISAAC RICALDE–relativo ao\nmês de DEZEMBRO de 2025, no valor de R$ 10.000,00":     "ISAAC RICALDE",
		"apresentada pelo LUCAS MUNIZ\nDE ALMEIDA – LUCAS MUNIZ, relativo ao mês de AGOSTO de\n2025, no valor de R$ 10.000,00":                       "LUCAS MUNIZ",
		"apresentada pelo NATAN DA\nSILVA FERREIRA –relativo ao mês de NOVEMBRO de 2025, no\nvalor de R$ 10.000,00":                                  "NATAN DA SILVA FERREIRA",
		"apresentada pelo ALÉCIO BREDA\nDIAS – Vereador LECINHO, –relativo ao mês de DEZEMBRO de\n2025, no valor de R$ 10.000,00":                    "LECINHO",
	} {
		if q, ok := ParliamentaryQuotaOf(ActPrestacaoContas, "CEAPM "+text); !ok || q.Councillor != want {
			t.Errorf("%q: esperava %q, veio %+v %v", text, want, q, ok)
		}
	}
	if _, ok := ParliamentaryQuotaOf(ActContrato, body); ok {
		t.Fatal("só prestação de contas tem cota parlamentar")
	}
	if _, ok := ParliamentaryQuotaOf(ActPrestacaoContas, "TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS da creche, no valor de R$ 1,00"); ok {
		t.Fatal("prestação de contas de creche não é cota parlamentar")
	}
}
