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
		{"global vence o mensal citado antes", ActContrato,
			"valor mensal de R$ 1.000,00, perfazendo o valor global de R$ 12.000,00.", 1200000},
		{"mensal sozinho ainda vale", ActContrato, "pelo valor mensal de R$ 1.000,00.", 100000},
		{"palavras com r entre o rótulo e o valor", ActContrato, "VALOR GLOBAL PARA O EXERCÍCIO: R$ 1.000,00", 100000},
		{"cabeçalho de tabela pula para o valor global", ActAta,
			"RAZÃO SOCIAL/NOME: LAZZARI MARTINEZ COMERCIO VAREJISTA Quantidade Valor Unitário Valor Total 200 R$ 6,58 R$ 1.316,00 100 R$ 2,71 R$ 271,00 Valor Global: R$ 1.587,00 3. VALIDADE DA ATA", 158700},
		{"cabeçalho de tabela sem outro valor fica sem valor", ActAta,
			"RAZÃO SOCIAL/NOME: MILLENIUM COMERCIO SERVIÇO LTDA Quantidade Valor Unitário Valor Total 200 R$ 46,74 R$ 9.348,00 3. VALIDADE DA ATA", 0},
		{"cabeçalho com unitário depois do total", ActAta,
			"UNID QUANT Unid 2940 Unid 172 MARCA VALOR VALOR TOTAL UNIT. Nacional R$ 290,00 R$ 852.600,00", 0},
		{"coluna de marca e valor registrado", ActAta,
			"Fornecimento de copos de 200 ml de água (cx. c/48). MARCA VALOR REGISTRADO AGUÁ SOL R$ 8,99 MONTANHA R$ 18,99", 0},
		{"cabeçalho e depois o total dos itens", ActLicitacao,
			"INCLUSIVE OPERADOR VALOR UNITÁRIO VALOR TOTAL R$ 40,00 R$ 211.200,00 R$ 120,00 R$ 950.400,00 VALOR TOTAL ITENS: R$ 1.161.600,00", 116160000},
		{"preço unitário citado antes do total", ActContrato,
			"ao preço unitário de R$ 5,00, com valor total de R$ 500,00.", 50000},
		{"marca do item antes do total", ActLicitacao,
			"01-01 UNID. 6,80 197 11.375,00 S/MARCA Valor Total R$ 1.567.580,00", 156758000},
		{"coluna de marca e valor total sem unitário", ActLicitacao,
			"precificados item por item. MARCA VALOR TOTAL YAMAHA DAS JBL SHURE PIONER RMV R$ 1.715.000,00", 171500000},
		{"rótulo verdadeiro logo depois do cabeçalho", ActLicitacao,
			"OG MED VALOR UNIT. VALOR TOTAL 4,790 9.580,00 Valor Total R$ 9.580,00", 958000},
		{"célula de tabela com outro valor emendado", ActDispensa,
			"R$ 12.870,00 R$ 4,99 VALOR TOTAL R$ 2.994,00 R$ 22.312,80 Valor Global: R$ 22.312,80 (vinte e dois mil)", 2231280},
		{"coluna de total final com valores seguidos", ActLicitacao,
			"UNITÁRIO FINAL R$ 1.253,50 VALOR TOTAL FINAL 30 500 20 R$ 20,87 R$ 20,86 R$ 11.473,00", 0},
		{"valor seguido de vírgula e outro valor continua valendo", ActContrato,
			"no valor global de R$ 12.000,00, sendo R$ 1.000,00 por mês.", 1200000},
		{"total com dois-pontos e valores emendados continua valendo", ActAta,
			"60 UNID. 20 VALOR TOTAL ITENS: R$ 12.288,37 R$ 240,85 R$ 3.612,75", 1228837},
		{"total sem cabeçalho de tabela continua valendo", ActAta,
			"Modelo: K31201Y Valor Total R$ 800,00 São Gonçalo, 26 de agosto de 2021.", 80000},
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
