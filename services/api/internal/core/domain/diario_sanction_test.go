package domain

import "testing"

func TestClassifyDiarioSanctionReadsRealPublications(t *testing.T) {
	cases := []struct {
		title, body string
		want        DiarioSanctionKind
	}{
		{"EXTRATO DE MULTA", "O SECRETÁRIO MUNICIPAL DE EDUCAÇÃO decide aplicar a sanção de multa à empresa Home Bread, CNPJ 00.768.165/0001-08", SanctionFine},
		{"EXTRATO DE ADVERTÊNCIA", "Fica, pela presente, ADVERTIDA a empresa RB Rio Comércio de Produtos Eireli", SanctionWarning},
		{"DESPACHO DE ADVERTÊNCIA", "Referência : 974/2022 Assunto : Descumprimento de Cláusula Contratual", SanctionWarning},
		{"EXTRATO DE SANÇÃO", "O SG-PREVI ... resolve aplicar a penalidade de impedimento de licitar e contratar à empresa X", SanctionDebarment},
		{"NOTIFICAÇÃO DE IMPOSIÇÃO DE PENALIDADE/RESCISÃO CONTRATUAL", "aplica a penalidade de suspensão temporária de participação em licitação", SanctionSuspension},
		{"EXTRATO DE DECISÃO", "Termo de Referência, APLICO a penalidade de multa especial de 10% (dez por cento) do valor total da Ata", SanctionFine},
		{"PORTARIA Nº 007/GABPREFEITO/ 2016", "RESOLVE: Art. 1º - Aplicar a penalidade de Advertência à FUNDAÇÃO BIO RIO, inscrita no CNPJ", SanctionWarning},
		{"DECRETO", "a empresa fica sujeita, e aplica-se a pena de declaração de inidoneidade para licitar", SanctionIneligibility},
	}
	for _, c := range cases {
		if got, ok := ClassifyDiarioSanction(c.title, c.body); !ok || got != c.want {
			t.Errorf("%s: %q %v, esperava %q", c.title, got, ok, c.want)
		}
	}
}

func TestClassifyDiarioSanctionIgnoresNoise(t *testing.T) {
	cases := [][2]string{
		{"benefício de anistia de taxas e multas de IPTU promovido pela", "HOMOLOGO o procedimento licitatório em favor da empresa MACRO"},
		{"PORTARIA Nº 30/2025", "além de indícios de irregularidades; V – aplicar penalidade de advertência, subsidiado pelas informações"},
		{"EXTRATO DO CONTRATO", "O contrato prevê multa de 2% em caso de atraso."},
	}
	for _, c := range cases {
		if got, ok := ClassifyDiarioSanction(c[0], c[1]); ok {
			t.Errorf("%s classificado como %q", c[0], got)
		}
	}
}
