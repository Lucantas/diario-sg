package parser

import (
	"strings"
	"testing"
)

func TestLicenseNoticePartyLinesBelongToTheNotice(t *testing.T) {
	acts := New().Parse(`ATOS DO PREFEITO
DESPACHO DO PRESIDENTE
INDEFERIR os seguintes processos: 517/2025, 519/2025.
FÁBIO CÉSAR DA SILVEIRA
Presidente da 2ª JARI
MICAL INVEST E PARTICIPAÇÕES LTDA
CNPJ: 58.452.347/0001-55
CONCESSÃO DE LICENÇA
MICAL INVEST E PARTICIPAÇÕES LTDA torna público que recebeu
da Secretaria Municipal de Meio Ambiente e Transportes a LICENÇA.`)

	if len(acts) != 2 || strings.Contains(acts[0].Body, "58.452.347") || !strings.Contains(acts[1].Body, "CNPJ: 58.452.347/0001-55") || acts[1].Title != "CONCESSÃO DE LICENÇA" {
		t.Fatalf("nome e CNPJ do aviso vão para o aviso: %+v", acts)
	}
}

func TestCNPJAtTheEndOfAnActStaysWhenTheNextActIsAboutSomeoneElse(t *testing.T) {
	acts := New().Parse(`ATOS DO PREFEITO
EXTRATO DO CONTRATO Nº 3/2026
Objeto: obras.
ESPETO CHIC PETISCARIA EIRELI
CNPJ 37.413.018/0001-25
TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS
Processo n.º 40.945/2022 Tendo em vista o que consta dos autos.`)

	if len(acts) != 2 || !strings.Contains(acts[0].Body, "37.413.018") {
		t.Fatalf("CNPJ fica no ato de cima: %+v", acts)
	}
}

func TestHeaderlessActAfterAKnownAcronymTakesTheOrgan(t *testing.T) {
	text := "Exonera:\na contar de 01 de agosto de 2016, FULANO, da função.\nPort. nº 1360/2016\n" +
		"SEMAD\nLicença Prêmio:\nMatr. 7480 FULANO DE TAL, professor docente I, a partir de 01/01/2018"

	want := "Port. nº 1360/2016=|SEMAD=SEMAD"
	if got := organs(text); got != want {
		t.Errorf("esperava\n%s\nveio\n%s", want, got)
	}
}

func TestPageCountAndSiteFooterAreNoise(t *testing.T) {
	acts := New().Parse(`ATOS DO PREFEITO
DECRETO Nº 1/2026
O PREFEITO MUNICIPAL DE SÃO GONÇALO, no uso de suas
https://www.saogoncalo.rj.gov.br/diario-oficial/
3
atribuições legais, decreta.
Continuação do D.O.E. em 17/04/2025
1/1
Nomeia:
a contar de 01 de abril de 2025, FULANO, para o cargo.
Port. nº 10/2025`)

	for _, a := range acts {
		if strings.Contains(a.Body, "saogoncalo.rj.gov.br") || a.Title == "1/1" || strings.Contains(a.Body, "\n3\n") {
			t.Fatalf("rodapé no ato: %+v", acts)
		}
	}
	if len(acts) != 2 || acts[1].Title != "Port. nº 10/2025" {
		t.Fatalf("atos: %+v", acts)
	}
}
