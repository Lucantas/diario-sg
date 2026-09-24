package domain

import "testing"

func TestModalityOf(t *testing.T) {
	cases := []struct {
		name  string
		typ   ActType
		title string
		body  string
		want  Modality
	}{
		{"extrato de contrato por dispensa", ActContrato, "EXTRATO DO CONTRATO Nº 5/2023",
			"Processo nº 5708/2023. Modalidade de licitação: Dispensa de Licitação nº 002/2023/SEMFA.", ModalityDispensa},
		{"inexigibilidade no tipo dispensa", ActDispensa, "EXTRATO DE INEXIGIBILIDADE DE LICITAÇÃO",
			"Ratifico a inexigibilidade de licitação, art. 74 da Lei 14.133.", ModalityInexigibilidade},
		{"aditivo de contrato do pregão", ActAditivo, "EXTRATO DO 2º TERMO ADITIVO",
			"Contrato decorrente do Pregão Eletrônico nº 90/2024. Adesão à ata não se aplica.", ModalityPregao},
		{"dispensa eletrônica", ActLicitacao, "AVISO DE DISPENSA ELETRÔNICA Nº 90003/2025", "Objeto: materiais.", ModalityDispensa},
		{"credenciamento", ActAditivo, "EXTRATO DO QUARTO TERMO ADITIVO AO CONTRATO DE ADESÃO MEDIANTE CREDENCIAMENTO Nº 44/2011", "", ModalityCredenciamento},
		{"sem modalidade citada", ActContrato, "EXTRATO DE CONTRATO TEMPORÁRIO Nº 292/2019", "Partes: Município e Fulana.", ""},
		{"portaria que cita pregão não tem modalidade", ActPortaria, "PORTARIA Nº 8/2024",
			"Altera a equipe de apoio da pregoeira na modalidade pregão.", ""},
	}
	for _, c := range cases {
		if got := ModalityOf(c.typ, c.title, c.body); got != c.want {
			t.Errorf("%s: esperava %q, veio %q", c.name, c.want, got)
		}
	}
}

func TestMainValueCents(t *testing.T) {
	cases := []struct {
		name string
		typ  ActType
		body string
		want int64
	}{
		{"valor global depois de itens", ActContrato,
			"Item 1: R$ 10,00. Valor global: R$ 50.000,00 (cinquenta mil reais).", 5000000},
		{"valor total em maiúsculas", ActDispensa, "VALOR TOTAL: R$ 57.999,60 (cinquenta e sete mil…)", 5799960},
		{"cota parlamentar", ActPrestacaoContas, "relativo ao mês de OUTUBRO de 2025, no valor de R$ 10.000,00 (dez mil reais).", 1000000},
		{"acréscimo do aditivo", ActAditivo, "Fica acrescido o valor, com acréscimo de R$ 12.500,00 ao contrato.", 1250000},
		{"sem expressão de valor principal", ActContrato, "Itens: R$ 1,00; R$ 2,00.", 0},
		{"tipo sem valor principal", ActNomeacao, "no valor de R$ 3.000,00", 0},
	}
	for _, c := range cases {
		if got := MainValueCents(c.typ, c.body); got != c.want {
			t.Errorf("%s: esperava %d, veio %d", c.name, c.want, got)
		}
	}
}

func TestModalityValid(t *testing.T) {
	if !ModalityInexigibilidade.Valid() || Modality("carta").Valid() || Modality("").Valid() {
		t.Fatal("só as modalidades conhecidas são válidas")
	}
}
