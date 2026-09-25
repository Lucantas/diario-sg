package domain

import "testing"

func TestSupplierNameOf(t *testing.T) {
	cases := []struct{ body, want string }{
		{"Processo: Processo n.º 2360/2022 Partes: Procuradoria Geral do Município de São Gonçalo X Lógica Tecnologia Ltda Objeto: Locação de Equipamentos de Informática",
			"Lógica Tecnologia Ltda"},
		{"Partes: Prefeitura Municipal de São Gonçalo – Procuradoria Geral e Secretaria de Fazenda x DBNOVA Tecnologia Ltda. Objeto: Locação de Sistema",
			"DBNOVA Tecnologia Ltda"},
		{"HOMOLOGO o correspondente procedimento licitatório em favor da empresa F.P. VIEIRA ENGENHARIA LTDA, CNPJ 14.180.324/0001-63",
			"F.P. VIEIRA ENGENHARIA LTDA"},
		{"conforme Decreto Municipal nº 099/2010. CONTRATADO: C. TEIXEIRA 110 COMÉRCIO DE ALIMENTOS LTDA VALOR: R$ 202.800,00",
			"C. TEIXEIRA 110 COMÉRCIO DE ALIMENTOS LTDA"},
		{"Partes: Município de São Gonçalo e a Empresa Consórcio Vicinais. Objeto: Segundo Termo Aditivo",
			"Consórcio Vicinais"},
		{"PARTES: FUNDAÇÃO DE ARTES, ESPORTE E LAZER DE SÃO GONÇALO – FAESG E A EMPRESA PARVAIM SOFTWARE DE GESTÃO LTDA. OBJETO: Contratação",
			"PARVAIM SOFTWARE DE GESTÃO LTDA"},
		{"Partes: Município de São Gonçalo e a Empresa Força Ambiental Ltda para execução dos serviços de Coleta",
			"Força Ambiental Ltda"},
		{"RECONHEÇO E RATIFICO a Dispensa Emergencial, Processo n.º 2047/21 fundamento no art. 24, inciso IV da Lei n.º 8.666/93, para contratação de serviço de locação",
			""},
	}
	for _, c := range cases {
		if got := SupplierNameOf(c.body); got != c.want {
			t.Errorf("%q\nveio %q, esperava %q", c.body, got, c.want)
		}
	}
}

func TestSupplierNameKey(t *testing.T) {
	same := [][]string{
		{"Lógica Tecnologia Ltda", "LOGICA TECNOLOGIA EIRELI", "Lógica Tecnologia Ltda."},
		{"DBNOVA Tecnologia Ltda", "DBNOVA TECNOLOGIA LTDA-EPP"},
		{"Construtora Marquise S/A.", "Construtora Marquise S/A", "CONSTRUTORA MARQUISE S.A."},
	}
	for _, names := range same {
		want := SupplierNameKey(names[0])
		for _, n := range names[1:] {
			if got := SupplierNameKey(n); got != want {
				t.Errorf("%q deu %q, esperava %q", n, got, want)
			}
		}
	}
	if SupplierNameKey("Lógica Tecnologia Ltda") != "logica tecnologia" || SupplierNameKey("LTDA") != "" {
		t.Errorf("chave inesperada: %q %q", SupplierNameKey("Lógica Tecnologia Ltda"), SupplierNameKey("LTDA"))
	}
}
