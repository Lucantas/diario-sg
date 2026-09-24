package domain

import "testing"

func TestPanelValueRoleOf(t *testing.T) {
	cases := []struct {
		name  string
		typ   ActType
		title string
		head  string
		want  PanelValueRole
	}{
		{"extrato de contrato", ActLicitacao, "EXTRATO DO CONTRATO 010/SEMED/2025 DO PREGÃO",
			"EXTRATO DO CONTRATO 010/SEMED/2025 DO PREGÃO\nELETRÔNICO PMSG- 90013/2025.\nProcesso: 7717/2025.\nPartes: MUNICÍPIO DE SÃO GONÇALO e F.P. VIEIRA ENGENHARIA",
			PanelValueContracted},
		{"ratificação de dispensa", ActDispensa, "EXTRATO DE RATIFICAÇÃO DE DISPENSA LICITAÇÃO",
			"EXTRATO DE RATIFICAÇÃO DE DISPENSA LICITAÇÃO\nPROCESSO ADMINSTRATIVO Nº 21577/19\nRATIFICO a situação de dispensa de licitação",
			PanelValueContracted},
		{"adesão a ata", ActContrato, "EXTRATO DE TERMO DE ADESÃO",
			"EXTRATO DE TERMO DE ADESÃO\nPROCESSO ADMINISTRATIVO N.º 6.013/2021\nO MUNICÍPIO DE SÃO GONÇALO, através da Secretaria\nMunicipal de Desenvolvimento Urbano, torna pública a adesão\nà Ata de Registro de Preços n.º 92/2020",
			PanelValueContracted},
		{"contrato de pregão SRP", ActLicitacao, "PREGÃO ELETRÔNICO SRP N.º 035/2021",
			"PREGÃO ELETRÔNICO SRP N.º 035/2021\nCONTRATO N.º 017/FMS/2021.\nPARTES: FUNDAÇÃO MUNICIPAL DE SAÚDE DE SÃO GONÇALO,",
			PanelValueContracted},
		{"ata de registro de preços", ActLicitacao, "EXTRATO DA ATA DE REGISTRO DE PREÇOS",
			"EXTRATO DA ATA DE REGISTRO DE PREÇOS\nO MUNICÍPIO DE SÃO GONÇALO torna público, para o\nconhecimento de todos os interessados, o Extrato da Ata de\nRegistro de Preços Nº 001/SEMMATRAN/2026",
			PanelValueRegistered},
		{"extrato trimestral", ActContrato, "EXTRATO TRIMESTRAL - O MUNICIPIO DE SÃO GONÇALO",
			"EXTRATO TRIMESTRAL - O MUNICIPIO DE SÃO GONÇALO\ntorna público para o conhecimento de todos os interessados,\no Extrato da Trimestral da Ata de Registro de Preços",
			PanelValueRegistered},
		{"homologação no título", ActLicitacao, "EXTRATO DA HOMOLOGAÇÃO DO PREGÃO ELETRÔNICO N°.",
			"EXTRATO DA HOMOLOGAÇÃO DO PREGÃO ELETRÔNICO N°.\n90013/2025.\nPROCESSO ADMINISTRATIVO Nº. 7717/2025.\nNos termos do relatório final apresentado pelo Pregoeiro",
			PanelValueNone},
		{"homologação só no texto", ActLicitacao, "PREGÃO ELETRÔNICO Nº 90030/2025",
			"PREGÃO ELETRÔNICO Nº 90030/2025\nPROCESSO ADMINISTRATIVO Nº 2.686/2025\nHomologo o resultado da licitação referente ao Pregão Eletrônico nº\n90030/2025",
			PanelValueNone},
		{"aviso de licitação", ActLicitacao, "AVISO DE LICITAÇÃO",
			"AVISO DE LICITAÇÃO\nPregão Presencial FMS SRP nº 009/2016.\nProcesso nº 2.888/2016.",
			PanelValueNone},
		{"aditivo", ActAditivo, "EXTRATO DO PRIMEIRO TERMO ADITIVO AO CONTRATO 010/SEMED/2025",
			"EXTRATO DO PRIMEIRO TERMO ADITIVO AO CONTRATO 010/SEMED/2025 DO PREGÃO ELETRÔNICO PMSG-90013/2025. Processo: 7.717/2025.",
			PanelValueNone},
	}
	for _, c := range cases {
		if got := PanelValueRoleOf(c.typ, c.title, c.head); got != c.want {
			t.Errorf("%s: veio %d, esperava %d", c.name, got, c.want)
		}
	}
}
