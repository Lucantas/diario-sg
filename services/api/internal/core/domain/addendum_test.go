package domain

import "testing"

func TestDeclaredIncreaseBasisPoints(t *testing.T) {
	cases := []struct {
		name  string
		typ   ActType
		title string
		body  string
		want  int
	}{
		{"equivale a 24,88%", ActAditivo, "EXTRATO DE TERMO ADITIVO",
			"Fica o Contrato PMSG n.º 012/2020 Certificado com reflexo financeiro, havendo um acréscimo R$ 363.730,94 que equivale a 24,88% do valor contratado, com base no artigo 65 da Lei n.º 8.666/93.", 2488},
		{"acréscimo equivalente a 46,37%", ActAditivo, "EXTRATO DE TERMO ADITIVO DE CONTRATO",
			"OBJETO: O presente Termo Aditivo tem por objetivo o acréscimo equivalente a 46,37% do valor inicial contratado, o que perfaz um total de R$ 240.802,53", 4637},
		{"percentual depois de citar a lei", ActAditivo, "EXTRATO DO PRIMEIRO TERMO ADITIVO QUANTITATIVO E DE PRORROGAÇÃO DO CONTRATO 001/SEMCON/2022.",
			"Objeto: Acréscimo quantitativo de itens específicos do contrato já firmado entre as partes, atendendo aos limites prescritos pelo § 1º do artigo 65 da Lei 8.666/93, em 8,63% (oito vírgula sessenta e três por cento) do valor global do contrato.", 863},
		{"acréscimo à 10%", ActAditivo, "EXTRATO DO PRIMEIRO TERMO ADITIVO DE CONTRATO",
			"OBJETO: Termo Aditivo de acréscimo equivalente à 10% (dez por cento) do valor inicial do contrato, correspondente à R$ 54.000,00", 1000},
		{"contrato com termo aditivo no título", ActContrato, "EXTRATO DO TERCEIRO TERMO ADITIVO",
			"Objeto: acréscimo de 25% no valor inicialmente firmado Fundamento: Art. 65, Inciso I, alínea b da Lei nº 8.666/93.", 2500},
		{"reajuste", ActAditivo, "EXTRATO DO SEGUNDO TERMO ADITIVO DE PRORROGAÇÃO DO CONTRATO 001/SEMCON/2022 COM REAJUSTE.",
			"Novo valor total: R$ 79.101.466,37, correspondente a um reajuste de 12,06% (doze vírgula seis por cento).", 0},
		{"acréscimo em aditivo de reajuste", ActAditivo, "EXTRATO DO QUINTO TERMO ADITIVO DO CONTRATO 001/SEMCON/2022 COM REAJUSTE.",
			"Novo valor total: R$ 90.301.041,52, correspondente a um acréscimo de 26,92% (vinte e seis vírgula noventa e dois por cento) do valor original do contrato.", 0},
		{"retificação", ActAditivo, "EXTRATO DE TERMO ADITIVO",
			"Processo administrativo nº 59.856/2021 Publicado no D.O.E. em 20/03/2023. Onde se lê: “fica o contrato nº 001/SEMDUR/2022 aditivado em R$3.232.568,41 correspondente a um acréscimo financeiro de aproximadamente 20,99%", 0},
		{"limite da lei", ActAditivo, "EXTRATO DE TERMO ADITIVO",
			"Objeto: acréscimo de itens, respeitado o limite de até 25% previsto no art. 65 da Lei 8.666/93.", 0},
		{"não é aditivo", ActContrato, "EXTRATO DE CONTRATO", "Objeto: acréscimo de 10% no valor.", 0},
	}
	for _, c := range cases {
		if got := DeclaredIncreaseBasisPoints(c.typ, c.title, c.body); got != c.want {
			t.Errorf("%s: veio %d, esperava %d", c.name, got, c.want)
		}
	}
}

func TestAddendumOrdinal(t *testing.T) {
	cases := map[string]int{
		"EXTRATO SEGUNDO TERMO ADITIVO":                       2,
		"Primeiro Termo aditivo ao Contrato n° 019/2016":      1,
		"EXTRATO DO 5º TERMO ADITIVO":                         5,
		"EXTRATO DO DÉCIMO TERMO ADITIVO":                     10,
		"EXTRATO DE TERMO ADITIVO":                            0,
		"EXTRATO DO SÉTIMO TERMO ADITIVO AO CONTRATO 01/2020": 7,
	}
	for text, want := range cases {
		if got := AddendumOrdinal(text, ""); got != want {
			t.Errorf("%q: veio %d, esperava %d", text, got, want)
		}
	}
}

func TestMentionsRenovation(t *testing.T) {
	if !MentionsRenovation("contratação de empresa de engenharia para reforma e construção de novas instalações") ||
		MentionsRenovation("aquisição de equipamentos") || MentionsRenovation("reformulação do contrato") {
		t.Error("MentionsRenovation")
	}
}
