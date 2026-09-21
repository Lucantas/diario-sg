package parser

import (
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
	acts := New().Parse("texto solto sem cabeçalho")
	if len(acts) != 1 || acts[0].Type != domain.ActOutro {
		t.Fatalf("esperava 1 ato 'outro', veio %+v", acts)
	}
}
