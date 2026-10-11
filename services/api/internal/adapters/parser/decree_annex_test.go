package parser

import (
	"strings"
	"testing"
)

const edition1738Decrees = `DECRETO N.º 359/2026
ABRE CRÉDITO SUPLEMENTAR E ALTERA O ORÇAMENTO DA SECRETARIA
MUNICIPAL DE DESENVOLVIMENTO URBANO.
O PREFEITO MUNICIPAL DE SÃO GONÇALO, no uso de suas
atribuições legais, DECRETA:
Art. 1º Fica aberto crédito suplementar, no valor de R$ 5.365.000,00.
São Gonçalo, 31 de julho de 2026.
NELSON RUAS DOS SANTOS
Prefeito

O PREFEITO MUNICIPAL DE SÃO GONÇALO, no uso de suas
atribuições legais, DECRETA:
Art. 1º Fica aberto crédito suplementar, no valor de R$ 3.898.000,00.
São Gonçalo, 31 de julho de 2026.
NELSON RUAS DOS SANTOS
Prefeito
ANEXO DECRETO Nº 361/2026
QUADRO DE DETALHAMENTO DA DESPESA - EXERCÍCIO 2026
TOTAL GERAL
5.365.000,00
DECRETO N.º 361/2026
ABRE CRÉDITO SUPLEMENTAR E ALTERA O ORÇAMENTO DA SECRETARIA
MUNICIPAL DE DESENVOLVIMENTO URBANO.
ANEXO DECRETO Nº 359/2026
QUADRO DE DETALHAMENTO DA DESPESA - EXERCÍCIO 2026
ACRÉSCIMO
3.898.000,00`

func TestDecreeTextAndAnnexMisplacedByColumnsGoBackToTheirDecree(t *testing.T) {
	acts := New().Parse(edition1738Decrees)

	if len(acts) != 2 {
		t.Fatalf("esperava 2 decretos, veio: %s", titles(acts))
	}
	d359, d361 := acts[0].Body, acts[1].Body
	if strings.Contains(d359, "R$ 3.898.000,00") || strings.Contains(d359, "ANEXO DECRETO Nº 361") {
		t.Errorf("o 359 ficou com o texto ou o anexo do 361: %q", d359)
	}
	if !strings.Contains(d359, "ANEXO DECRETO Nº 359") {
		t.Errorf("o anexo do 359 não voltou para ele: %q", d359)
	}
	if !strings.Contains(d361, "R$ 3.898.000,00") || !strings.Contains(d361, "ANEXO DECRETO Nº 361") {
		t.Errorf("o 361 não recebeu o próprio texto e anexo: %q", d361)
	}
	if strings.Contains(d361, "ANEXO DECRETO Nº 359") {
		t.Errorf("o 361 ficou com o anexo do 359: %q", d361)
	}
}

func TestDecreeKeepsItsOwnPreambleWhenAForeignAnnexFollows(t *testing.T) {
	acts := New().Parse(`DECRETO Nº 087/2024
ABRE CRÉDITO SUPLEMENTAR.
O PREFEITO MUNICIPAL DE SÃO GONÇALO, no uso de suas atribuições legais, DECRETA:
Art. 1º Fica aberto crédito suplementar no valor de R$ 10.000,00.
NELSON RUAS DOS SANTOS
Prefeito
ANEXO DECRETO Nº 085/2024
TOTAL GERAL
20.000,00
DECRETO Nº 085/2024
ABRE CRÉDITO SUPLEMENTAR.
O PREFEITO MUNICIPAL DE SÃO GONÇALO, no uso de suas atribuições legais, DECRETA:
Art. 1º Fica aberto crédito suplementar no valor de R$ 20.000,00.
NELSON RUAS DOS SANTOS
Prefeito`)

	if len(acts) != 2 {
		t.Fatalf("esperava 2 decretos, veio: %s", titles(acts))
	}
	if !strings.Contains(acts[0].Body, "R$ 10.000,00") || strings.Contains(acts[0].Body, "ANEXO DECRETO Nº 085") {
		t.Errorf("o 087 devia manter o próprio texto e perder o anexo do 085: %q", acts[0].Body)
	}
	if !strings.HasSuffix(acts[1].Body, "TOTAL GERAL\n20.000,00") {
		t.Errorf("o anexo do 085 devia ir para o fim dele: %q", acts[1].Body)
	}
}

func TestAnnexOfADecreeOutsideTheEditionStaysPut(t *testing.T) {
	acts := New().Parse(`DECRETO Nº 120/2026
ALTERA O ANEXO DO DECRETO Nº 119/2026.
O PREFEITO MUNICIPAL DE SÃO GONÇALO, no uso de suas atribuições legais, DECRETA:
Art. 1º O anexo passa a vigorar conforme abaixo.
ANEXO DECRETO Nº 119/2026
QUADRO NOVO`)

	if len(acts) != 1 || !strings.Contains(acts[0].Body, "ANEXO DECRETO Nº 119/2026") {
		t.Fatalf("anexo de decreto que não está na edição fica onde está: %+v", acts)
	}
}
