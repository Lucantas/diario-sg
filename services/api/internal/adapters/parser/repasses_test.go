package parser

import (
	"strings"
	"testing"
)

const edition1784Repasses = `DESPACHO DO SECRETÁRIO
PROCESSO SEI nº 03.23295/2026-0 - DEFIRO licença por motivo de
doença em pessoa da família à servidora, por 10 (dez) dias.
São Gonçalo, 06 de outubro de 2026.
DANIEL LIMA DE MAGALHÃES BASTOS
Secretário Municipal de Administração

SEMFA
REPASSES FEDERAIS
SETEMBRO - 2026
DATA
10/09/2026
18/09/2026
ORGÃO DE
VALOR (R$)
REPASSE
GOVERNO
FPM
R$ 9.877.551,58
FEDERAL
GOVERNO
FPM
R$ 2.170.638,79
FEDERAL
Obs: Os Repasses São Considerados Pelos Valores Brutos
RODRIGO TORREGROSA OLIVEIRA
Secretário Municipal de Fazenda`

func TestFederalTransfersTableIsItsOwnActUnderSEMFA(t *testing.T) {
	acts := New().Parse(edition1784Repasses)

	if len(acts) != 2 {
		t.Fatalf("esperava o despacho e a tabela de repasses separados, veio: %s", titles(acts))
	}
	if strings.Contains(acts[0].Body, "9.877.551,58") {
		t.Errorf("o despacho engoliu a tabela de repasses: %q", acts[0].Body)
	}
	if acts[1].Title != "REPASSES FEDERAIS" || acts[1].Organ != "SEMFA" {
		t.Errorf("esperava REPASSES FEDERAIS (SEMFA), veio %q (%s)", acts[1].Title, acts[1].Organ)
	}
}

func TestTransfersMentionedInsideProseDoNotOpenAnAct(t *testing.T) {
	acts := New().Parse(`DECRETO Nº 100/2026
ABRE CRÉDITO SUPLEMENTAR COM BASE NOS
REPASSES FEDERAIS DO FUNDO NACIONAL DE SAÚDE.
O PREFEITO MUNICIPAL DE SÃO GONÇALO, no uso de suas atribuições legais, decreta.`)

	if len(acts) != 1 {
		t.Fatalf("REPASSES FEDERAIS no meio do título não abre ato: %s", titles(acts))
	}
}

func TestTransfersHeaderVariants(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"REPASSES FEDERAIS", true},
		{"REPASSES FEDERAIS – MAIO 2025", true},
		{"REPASSES FEDERAIS - SETEMBRO/2026", true},
		{"REPASSES ESTADUAIS", true},
		{"REPASSES FEDERAIS DO FUNDO NACIONAL DE SAÚDE.", false},
		{"Repasses Federais", false},
	}
	for _, c := range cases {
		if got := isHeader(c.line); got != c.want {
			t.Errorf("isHeader(%q) = %v, esperava %v", c.line, got, c.want)
		}
	}
}

func TestTransfersTableWithoutAcronymBelongsToSEMFA(t *testing.T) {
	acts := New().Parse(`DECRETO Nº 100/2025
O PREFEITO MUNICIPAL DE SÃO GONÇALO, no uso de suas atribuições legais, decreta.

FAELSG
CHAMAMENTO PÚBLICO PARA FINS DE CREDENCIAMENTO DE PROFISSIONAIS.
A Fundação de Artes, Esporte e Lazer torna público o credenciamento de instrutores.
REPASSES FEDERAIS
MARÇO - 2025
GOVERNO
FPM
R$ 2.303.323,83
FEDERAL
Secretário Municipal de Fazenda`)

	if len(acts) != 3 {
		t.Fatalf("esperava o decreto, o chamamento e a tabela, veio: %s", titles(acts))
	}
	if acts[1].Organ != "FAELSG" || acts[2].Organ != "SEMFA" {
		t.Errorf("órgãos: esperava FAELSG e SEMFA, veio %s e %s", acts[1].Organ, acts[2].Organ)
	}
}
