package parser

import (
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const sample = `PREFEITURA MUNICIPAL DE SÃO GONÇALO
DECRETO Nº 100/2026
Dispõe sobre o horário de funcionamento das repartições.

PORTARIA Nº 2.345/2026
O PREFEITO resolve NOMEAR Fulana de Tal para o cargo em comissão de Assessora.

PORTARIA Nº 2.346/2026
Resolve EXONERAR Beltrano da Silva do cargo de Diretor.

EXTRATO DO CONTRATO Nº 55/2026
Contratada: Empresa Exemplo LTDA. Valor: R$ 120.000,00.

EXTRATO DO TERMO ADITIVO Nº 3 AO CONTRATO Nº 12/2025
Prorrogação de prazo por 12 meses.

AVISO DE LICITAÇÃO - PREGÃO ELETRÔNICO Nº 10/2026
Objeto: aquisição de medicamentos.

DISPENSA DE LICITAÇÃO Nº 7/2026
Objeto: locação de imóvel.`

func TestParse(t *testing.T) {
	acts := New().Parse(sample)
	want := []domain.ActType{
		domain.ActDecreto, domain.ActNomeacao, domain.ActExoneracao,
		domain.ActContrato, domain.ActAditivo, domain.ActLicitacao, domain.ActDispensa,
	}
	if len(acts) != len(want) {
		t.Fatalf("esperava %d atos, veio %d: %+v", len(want), len(acts), acts)
	}
	for i, w := range want {
		if acts[i].Type != w {
			t.Errorf("ato %d (%q): esperava %s, veio %s", i, acts[i].Title, w, acts[i].Type)
		}
		if acts[i].Position != i {
			t.Errorf("ato %d: posição %d", i, acts[i].Position)
		}
	}
}

func TestParse_NoHeaders(t *testing.T) {
	acts := New().Parse("Texto solto sem nenhum cabeçalho reconhecido, mas com conteúdo suficiente para ser um ato.")
	if len(acts) != 1 || acts[0].Type != domain.ActOutro {
		t.Fatalf("esperava 1 ato 'outro', veio %+v", acts)
	}
}

// Formato das edições até abril de 2021: sem "ATOS DO PREFEITO", o anexo de
// pessoal vem logo após "GABINETE DO PREFEITO" e cada portaria é "verbo,
// corpo, Port. nº". "Continuação do D.O.E." abre um bloco novo no topo da
// página e a sigla do órgão abre a seção seguinte.
const oldFormatSample = `PREFEITURA
MUNICIPAL DE
SÃO GONÇALO
DIÁRIO OFICIAL ELETRÔNICO
Em, 18 de junho de 2020.
GABINETE DO PREFEITO
Exonera:
a contar de 17 de junho de 2020, ADÉLIA FICTÍCIA DE ALMEIDA
SOUZA – Mat.: 123102, do cargo em comissão de Assessor I.
Port. nº 808/2020
Nomeia:
a contar de 17 de junho de 2020, PATRÍCIA EXEMPLO MONTEIRO
– CPF: 026.***.***-10 para exercer o cargo em comissão de Assessor I.
Port. nº 809/2020

7
https://do.pmsg.rj.gov.br/

Continuação do D.O.E. em 18/06/2020
Designa:
a contar de 01 de julho de 2020, NEI FICTÍCIO RAMALHO
FILHO - Mat.: 123443, para responder pelo cargo de Diretor.
Port. nº 827/2020

SEMFA
EXTRATO DE CONTRATO DE PRESTAÇÃO DE SERVIÇOS
Partes: MUNICÍPIO DE SÃO GONÇALO e OBJECTTI SOLUÇÕES LTDA.
Objeto: Aquisição de 72 (setenta e dois) certificados digitais.`

func TestParse_OldFormatTrailingPortariaNumber(t *testing.T) {
	acts := New().Parse(oldFormatSample)
	want := []struct {
		title string
		typ   domain.ActType
		first string
	}{
		{"Port. nº 808/2020", domain.ActExoneracao, "Exonera:"},
		{"Port. nº 809/2020", domain.ActNomeacao, "Nomeia:"},
		{"Port. nº 827/2020", domain.ActPortaria, "Designa:"},
		{"EXTRATO DE CONTRATO DE PRESTAÇÃO DE SERVIÇOS", domain.ActContrato, "EXTRATO DE CONTRATO DE PRESTAÇÃO DE SERVIÇOS"},
	}
	if len(acts) != len(want) {
		t.Fatalf("esperava %d atos, veio %d: %+v", len(want), len(acts), acts)
	}
	for i, w := range want {
		a := acts[i]
		if a.Title != w.title || a.Type != w.typ || a.Position != i || !strings.HasPrefix(a.Body, w.first) {
			t.Errorf("ato %d: esperava %q/%s começando com %q, veio %q/%s: %q", i, w.title, w.typ, w.first, a.Title, a.Type, a.Body)
		}
		if strings.Contains(a.Body, "GABINETE DO PREFEITO") || strings.Contains(a.Body, "Continuação") || strings.Contains(a.Body, "SEMFA") {
			t.Errorf("ato %d carrega mobília de página/seção: %q", i, a.Body)
		}
	}
	if !strings.Contains(acts[0].Body, "ADÉLIA FICTÍCIA") || strings.Contains(acts[0].Body, "PATRÍCIA") {
		t.Errorf("a portaria 808 deve conter só a exoneração de Adélia: %q", acts[0].Body)
	}
}

func TestParse_TrailerWithoutBodyKeepsTheNumber(t *testing.T) {
	acts := New().Parse("Port. nº 5/2020\n")
	if len(acts) != 1 || acts[0].Title != "Port. nº 5/2020" || acts[0].Type != domain.ActPortaria {
		t.Fatalf("esperava só a portaria 5/2020, veio %+v", acts)
	}
}
