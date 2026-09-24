package pdf

import (
	"os"
	"strings"
	"testing"
)

const twoColumnPage = `                 PODER LEGISLATIVO
                                         LEI MUNICIPAL 855/2018 DE 05/07/2018
 TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS                         TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS
  PARLAMENTAR   MUNICIPAL- CEAPM OUTUBRO 2025
____________________________________________________                PARLAMENTAR   MUNICIPAL- CEAPM OUTUBRO 2025
JUAN OLIVEIRA                                                     VEREADOR NELSINHO

Processo n: 1198/2025                                             Processo n: 1189/2025
apresentada pelo a JUAN PATRICK PINHEIRO DE OLIVEIRA              apresentada pelo a NELSON RUAS DOS SANTOS FILHO
mil reais).                                                                São Gonçalo, 02 de fevereiro de 2026.
         São Gonçalo, 03 de fevereiro de 2026.                                   PIERO CABRAL
                  PIERO CABRAL
                                                                                         Página 1 de 3`

func TestUntangleColumnsReadsTheLeftColumnBeforeTheRight(t *testing.T) {
	got, ok := untangleColumns(twoColumnPage)

	if !ok {
		t.Fatal("a página tem duas colunas")
	}
	want := strings.Join([]string{
		"PODER LEGISLATIVO",
		"LEI MUNICIPAL 855/2018 DE 05/07/2018",
		"TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS",
		"PARLAMENTAR MUNICIPAL- CEAPM OUTUBRO 2025",
		"____________________________________________________",
		"JUAN OLIVEIRA",
		"",
		"Processo n: 1198/2025",
		"apresentada pelo a JUAN PATRICK PINHEIRO DE OLIVEIRA",
		"mil reais).",
		"São Gonçalo, 03 de fevereiro de 2026.",
		"PIERO CABRAL",
		"TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS",
		"PARLAMENTAR MUNICIPAL- CEAPM OUTUBRO 2025",
		"VEREADOR NELSINHO",
		"",
		"Processo n: 1189/2025",
		"apresentada pelo a NELSON RUAS DOS SANTOS FILHO",
		"São Gonçalo, 02 de fevereiro de 2026.",
		"PIERO CABRAL",
		"Página 1 de 3",
	}, "\n")
	if got != want {
		t.Fatalf("colunas desentrelaçadas erradas:\n%s\n--- esperado:\n%s", got, want)
	}
}

func TestUntangleColumnsLeavesSingleColumnPagesAlone(t *testing.T) {
	page := `                 PODER LEGISLATIVO
RESOLUÇÃO Nº 12/2026
A MESA DIRETORA da Câmara Municipal de São Gonçalo, no uso de suas atribuições legais,
resolve nomear FULANO DE TAL para o cargo de assessor parlamentar.
                                                        São Gonçalo, 3 de fevereiro de 2026.`

	if _, ok := untangleColumns(page); ok {
		t.Fatal("página de uma coluna não deveria ser desentrelaçada")
	}
}

func TestSingleColumnLineBetweenColumnBlocksKeepsItsPlace(t *testing.T) {
	page := `Processo n: 1/2026                                                Processo n: 2/2026
Tendo em vista o parecer do vereador A                            Tendo em vista o parecer do vereador B
no valor de R$ 10.000,00.                                         no valor de R$ 9.000,00.
RESOLUÇÃO Nº 13/2026 que atravessa a página inteira sem vão algum entre as duas metades da linha
Processo n: 3/2026                                                Processo n: 4/2026`

	got, ok := untangleColumns(page)

	if !ok {
		t.Fatal("a página tem duas colunas")
	}
	lines := strings.Split(got, "\n")
	want := []string{
		"Processo n: 1/2026", "Tendo em vista o parecer do vereador A", "no valor de R$ 10.000,00.",
		"Processo n: 2/2026", "Tendo em vista o parecer do vereador B", "no valor de R$ 9.000,00.",
		"RESOLUÇÃO Nº 13/2026 que atravessa a página inteira sem vão algum entre as duas metades da linha",
		"Processo n: 3/2026", "Processo n: 4/2026",
	}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("ordem errada:\n%s", got)
	}
}

func TestMergePagesUsesTheUntangledPageOnlyWhereThereAreColumns(t *testing.T) {
	raw := "capa\nlinha crua\f" + "página 2 crua"
	layout := "capa\nlinha crua\f" + twoColumnPage

	got := mergePages(raw, layout)

	pages := strings.Split(got, "\f")
	if len(pages) != 2 || pages[0] != "capa\nlinha crua" || !strings.HasSuffix(pages[1], "Página 1 de 3") {
		t.Fatalf("páginas misturadas erradas: %q", got)
	}
}

func TestMergePagesFallsBackToRawWhenPageCountsDiffer(t *testing.T) {
	raw := "uma página só"

	if got := mergePages(raw, "a\fb"); got != raw {
		t.Fatalf("deveria ficar com o texto cru: %q", got)
	}
}

func TestLeftLineEndingRightBeforeTheGutterStaysInTheLeftColumn(t *testing.T) {
	left, right := strings.Repeat("a", 60), strings.Repeat("d", 55)
	page := strings.Join([]string{
		left + "      " + right + " 1",
		left + "      " + right + " 2",
		left + "      " + right + " 3",
		left + "  ",
		strings.Repeat("b", 64),
	}, "\n")

	got, ok := untangleColumns(page)

	if !ok || strings.Count(got, "\n") != 7 || !strings.HasSuffix(got, right+" 3") {
		t.Fatalf("linhas curtas da esquerda deveriam ficar na coluna da esquerda: %q", got)
	}
}

func TestStaggeredColumnsAreSeparated(t *testing.T) {
	page, err := os.ReadFile("testdata/staggered.txt")
	if err != nil {
		t.Fatal(err)
	}

	got, ok := untangleColumns(string(page))

	if !ok {
		t.Fatal("colunas defasadas meia linha também são duas colunas")
	}
	rodrigo, cacau := strings.Index(got, "Processo n: 562/2025"), strings.Index(got, "Processo n: 566/2025")
	firstBody := got[rodrigo:cacau]
	if rodrigo < 0 || cacau < rodrigo || strings.Contains(firstBody, "CLAUDIO") || !strings.Contains(firstBody, "RODRIGO DUARTE, relativo ao") {
		t.Fatalf("o termo do vereador Rodrigo Duarte ficou misturado:\n%s", got)
	}
}
