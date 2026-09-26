package parser

import (
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const edition1257 = `ATOS DO PREFEITO
SEMFA
PORTARIA - SEI N.º 21/SEMFA/GAB/2024
INSTITUI A COMISSÃO PERMANENTE DE SINDICÂNCIA.
São Gonçalo, 22 de outubro de 2024.
RANDHAL JULIANO BARRETO COELHO
Secretário Municipal de Fazenda
SECRETARIA MUNICIPAL DE FAZENDA
AUTO DE INFRAÇÃO Nº 2000/2024
NOME: MG USINAGEN LTDA -ME
O contribuinte declarou mas não recolheu o ISS através do DAS.
PEDRO LUCIANO DE LEMOS FRANCO- mat. 13.744

SEMED

2
https://do.pmsg.rj.gov.br/

` + "\f" + `DIÁRIO OFICIAL

TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS COM
RESSALVA
apresentada pela CRECHE COMUNITÁRIA no valor de R$137.227,70.
MAURICIO NASCIMENTO DE ALMEIDA
Secretário Municipal de Educação

SEMTRAN
DESIGNAÇÃO DE FISCAIS PARA O CONTRATO N.º 02/SEMTRAN/24
Partes: a empresa LM CURSOS DE TRANSITO - CNPJ nº: 18.657.198/0001-46.
FABIO RICARDO FONTES LEMOS`

func TestActBoundariesOfEdition1257(t *testing.T) {
	acts := New().Parse(edition1257)

	want := []struct{ title, organ string }{
		{"PORTARIA - SEI N.º 21/SEMFA/GAB/2024", "SEMFA"},
		{"AUTO DE INFRAÇÃO Nº 2000/2024", "SEMFA"},
		{"TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS COM RESSALVA", "SEMED"},
		{"DESIGNAÇÃO DE FISCAIS PARA O CONTRATO N.º 02/SEMTRAN/24", "SEMTRAN"},
	}
	if len(acts) != len(want) {
		t.Fatalf("esperava %d atos, veio %d: %+v", len(want), len(acts), acts)
	}
	for i, w := range want {
		if acts[i].Title != w.title || acts[i].Organ != w.organ {
			t.Errorf("ato %d: esperava %q (%s), veio %q (%s)", i, w.title, w.organ, acts[i].Title, acts[i].Organ)
		}
	}
	if strings.Contains(acts[2].Body, "LM CURSOS") {
		t.Errorf("a prestação de contas da SEMED engoliu a designação da SEMTRAN: %q", acts[2].Body)
	}
}

func TestBareDesignacaoInsideATitleIsNotAnAct(t *testing.T) {
	acts := New().Parse(`PORTARIA N.º 005/SEMDUR/2026
DISPÕE SOBRE A
DESIGNAÇÃO
DO GESTOR PARA CONTRATAÇÃO DE SERVIÇOS DE ENGENHARIA.`)

	if len(acts) != 1 {
		t.Fatalf("DESIGNAÇÃO no meio do título não abre ato: %+v", acts)
	}
}

func TestProductExtractInsideAPriceListIsNotAnAct(t *testing.T) {
	acts := New().Parse(`EXTRATO DA ATA DE REGISTRO DE PREÇOS Nº 012/SEMED/2019
O MUNICÍPIO DE SÃO GONÇALO torna público o extrato da ata.
ITEM ESPECIFICAÇÃO
EXTRATO DE TOMATE –
embalagem de 300g, deve estar isento de fermentação.
EXTRATO DE ALOE VERA.
EXTRATO BANCÁRIO
VALOR TOTAL R$ 5.279.936,30`)

	if len(acts) != 1 {
		t.Fatalf("produto da lista não abre ato: %+v", acts)
	}
}

func TestHeaderWithCircumflexTypoOpensAnAct(t *testing.T) {
	acts := New().Parse(`EXTRATO DO 1º TERMO ADITIVO A ATA DE REGISTRO DE
PREÇOS Nº 003/SEMAS/2025
PARTES: MUNICÍPIO DE SÃO GONÇALO e TOP 01 COMERCIO E SERVIÇOS LTDA.
FELIPPE MATTOS MONTEIRO
SEMDUR
HOMOLOGAÇÂO/ADJUDICAÇÃO
CONCORRÊNCIA PÚBLICA/ ELETRÔNICA Nº 90010/2025.
PROCESSO ADMINISTRATIVO Nº. 27.298/2025.
no valor total de R$ 130.413.999,00`)

	if len(acts) != 2 || acts[1].Type != domain.ActLicitacao || !strings.Contains(acts[1].Body, "130.413.999,00") {
		t.Fatalf("HOMOLOGAÇÂO com acento trocado abre ato de licitação: %+v", acts)
	}
}

func TestKnownOrganFollowedByUppercaseTitleOpensAnAct(t *testing.T) {
	acts := New().Parse(`PORTARIA N.º 010/SEMED/2020
Designa servidores para a comissão de avaliação das unidades escolares.
SEMSADC
INFORMATIVO CORONAVÍRUS N.º 45/2020
A Secretaria Municipal de Saúde informa os casos confirmados no município.`)

	if len(acts) != 2 || acts[1].Title != "INFORMATIVO CORONAVÍRUS N.º 45/2020" || acts[1].Organ != "SEMSADC" {
		t.Fatalf("sigla de órgão seguida de título em caixa alta abre ato: %+v", acts)
	}
}

func TestUppercaseWordsInsideABodyDoNotOpenActs(t *testing.T) {
	acts := New().Parse(`EXTRATO DE CONTRATO Nº 010/2022
PARTES: MUNICÍPIO DE SÃO GONÇALO e EMPRESA X LTDA.
OBJETO
AQUISIÇÃO DE MATERIAL DE LIMPEZA PARA AS ESCOLAS
SEMED
SEMFA SEMGIPE SEMDUR 50 50 1
VALOR GLOBAL: R$ 1.000,00`)

	if len(acts) != 1 {
		t.Fatalf("palavra em caixa alta que não é órgão e lista de siglas não abrem ato: %+v", acts)
	}
}

func TestOrganFullNameBetweenAcronymAndHeaderIsNotAnAct(t *testing.T) {
	acts := New().Parse(`PORTARIA N.º 010/SMTC/2026
Designa servidores para a comissão de seleção do carnaval.
SMTC
SECRETARIA MUNICIPAL DE TURISMO E CULTURA
SMTC COM AMPARO NO FUNDO MUNICIPAL DE CULTURA
CHAMAMENTO PÚBLICO Nº 04/2026
Seleção de propostas culturais para o carnaval de 2027.`)

	if len(acts) != 2 || acts[1].Organ != "SMTC" || acts[1].Type != domain.ActLicitacao {
		t.Fatalf("nome do órgão por extenso não vira ato: %+v", acts)
	}
}

func TestAcronymCellInsideATableDoesNotOpenAnAct(t *testing.T) {
	acts := New().Parse(`Exonera:
a contar de 27 de dezembro de 2010, os servidores abaixo relacionados, da Secretaria Municipal de Governo.
SUBSECRETARIO
DAS-1
SSM
DIRETOR DE DIVISAO
DAS-5
SUPERVISOR
DAS-1
Port. nº 3165/2010`)

	if len(acts) != 1 || acts[0].Type != domain.ActExoneracao {
		t.Fatalf("célula de tabela com sigla não abre ato: %+v", acts)
	}
}

func TestLoneSignatureAfterAnOrganIsNotAnAct(t *testing.T) {
	acts := New().Parse(`PORTARIA N.º 010/SEMAD/2024
Designa servidores para a comissão de avaliação das unidades escolares.
SEMAD
LEONARDO NEVES DOS SANTOS DE OLIVEIRA
Secretário Municipal De Administração`)

	if len(acts) != 1 {
		t.Fatalf("assinatura solta não vira ato: %+v", acts)
	}
}

func TestEachExpenseAuthorizationIsItsOwnAct(t *testing.T) {
	acts := New().Parse(`AUTORIZAÇÃO DA DESPESA E ADJUDICAÇÃO
Processo nº 7405/2026
Eu, Júlia Carvalho Silva Sobreira, Secretária Municipal de Turismo e
Cultura, autorizo a contratação direta da empresa abaixo qualificada.
Contratada: VITROS LIMPEZAS DE VIDROS
Valor total: R$ 53.000,00 (cinquenta e três mil reais)
JULIA SOBREIRA
Secretária Municipal de Turismo e Cultura
AUTORIZAÇÃO DA DESPESA E ADJUDICAÇÃO
Processo nº 07537/2026
Eu, Júlia Carvalho Silva Sobreira, Secretária Municipal de Turismo e
Cultura, autorizo a contratação direta da empresa abaixo qualificada.
Contratada: R R Cores de Minas Tintas e Acessórios Ltda.
Valor total: R$ 63.232,50 (sessenta e três mil duzentos e trinta e dois reais)`)

	if len(acts) != 2 {
		t.Fatalf("cada autorização de despesa é um ato: %+v", acts)
	}
	if strings.Contains(acts[0].Body, "07537/2026") || !strings.Contains(acts[1].Body, "07537/2026") {
		t.Errorf("o segundo processo ficou no primeiro ato: %q", acts[0].Body)
	}
}
