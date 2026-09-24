package parser

import (
	"strings"
	"testing"
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
