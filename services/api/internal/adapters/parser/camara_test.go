package parser

import (
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const camaraHeader = "PODER LEGISLATIVO\nCÂMARA MUNICIPAL DE SÃO GONÇALO\nSão Gonçalo, 3 de novembro de 2025\n" +
	"Ano-08 / Edição - 138\n\n__________________________________________________________________\n" +
	"DIÁRIO OFICIAL ELETRÔNICO – D.O.E\nLEI MUNICIPAL 855/2018 DE 05/07/2018.\n"

func camaraEdition() string {
	return camaraHeader +
		"PORTARIA Nº 156/2025\nO PRESIDENTE DA CÂMARA MUNICIPAL DE SÃO GONÇALO, NO USO DE SUAS ATRIBUIÇÕES LEGAIS, RESOLVE:\n" +
		"EXONERAR A CONTAR DE 1º DE NOVEMBRO DE 2025 o servidor do gabinete.\n" +
		"GABINETE DO PRESIDENTE DA CÂMARA MUNICIPAL DE\nSÃO GONÇALO\n" +
		"São Gonçalo, 30 de outubro de 2025.\nPágina 1 de 2\n\f" +
		strings.Replace(camaraHeader, "Edição - 138", "Edição 138", 1) +
		"TERMO DE HOMOLOGAÇÃO\nHomologo o pregão eletrônico nº 4/2025 da Câmara.\n" +
		"Página 2 de 2\n"
}

func TestCamaraParserDropsPageHeadersAndFooters(t *testing.T) {
	acts := ForSource(domain.SourceDiarioCamara).Parse(camaraEdition())

	got := pageSpans(acts)
	want := []string{"PORTARIA Nº 156/2025 1-1", "TERMO DE HOMOLOGAÇÃO 2-2"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("esperava %v, veio %v", want, got)
	}
	for _, a := range acts {
		for _, noise := range []string{"PODER LEGISLATIVO", "Ano-08", "LEI MUNICIPAL 855/2018", "Página 1 de 2"} {
			if strings.Contains(a.Body, noise) {
				t.Errorf("%s ficou com %q no corpo", a.Title, noise)
			}
		}
	}
}

func TestCamaraParserKeepsSigningLinesInsideActs(t *testing.T) {
	acts := ForSource(domain.SourceDiarioCamara).Parse(camaraEdition())

	body := acts[0].Body
	for _, kept := range []string{"São Gonçalo, 30 de outubro de 2025.", "GABINETE DO PRESIDENTE DA CÂMARA MUNICIPAL DE\nSÃO GONÇALO"} {
		if !strings.Contains(body, kept) {
			t.Errorf("corpo perdeu %q:\n%s", kept, body)
		}
	}
}

func TestCamaraParserClassifiesActs(t *testing.T) {
	acts := ForSource(domain.SourceDiarioCamara).Parse(camaraEdition())

	if acts[0].Type != domain.ActExoneracao || acts[1].Type != domain.ActLicitacao {
		t.Errorf("tipos inesperados: %s, %s", acts[0].Type, acts[1].Type)
	}
}

func TestCamaraPageContinuationStaysInTheSameAct(t *testing.T) {
	text := camaraHeader + "RESOLUÇÃO Nº 885/2023\nEMENTA: CONCEDE O TÍTULO DE CIDADÃO GONÇALENSE.\nPágina 1 de 2\n\f" +
		camaraHeader + "Art. 2º Esta resolução entra em vigor na data de sua publicação.\n"

	got := pageSpans(ForSource(domain.SourceDiarioCamara).Parse(text))

	if len(got) != 1 || got[0] != "RESOLUÇÃO Nº 885/2023 1-2" {
		t.Errorf("esperava a resolução nas páginas 1-2, veio %v", got)
	}
}

func TestCamaraEditionNumber(t *testing.T) {
	cases := map[string]string{
		"São Gonçalo, 3 de novembro de 2025\nAno-08 / Edição - 138\n": "138",
		"Ano-03 / Edição – 128\n":                                     "128",
		"Ano-08 / Edição 138\n":                                       "138",
		"Ano-09 / Edição – 7.\n":                                      "7",
		"texto sem cabeçalho":                                         "",
	}
	for text, want := range cases {
		if got := ForSource(domain.SourceDiarioCamara).EditionNumber(text); got != want {
			t.Errorf("EditionNumber(%q) = %q, esperava %q", text, got, want)
		}
	}
}

func TestPrefeituraParserKeepsCamaraLikeLines(t *testing.T) {
	text := "DECRETO Nº 1/2026\nPODER LEGISLATIVO\nPágina 1 de 2\n"

	acts := ForSource(domain.SourceDiarioPrefeitura).Parse(text)

	if len(acts) != 1 || !strings.Contains(acts[0].Body, "PODER LEGISLATIVO\nPágina 1 de 2") {
		t.Errorf("a Prefeitura não deveria tirar linhas da Câmara: %+v", acts)
	}
}

func TestSetChoosesTheParserBySource(t *testing.T) {
	text := "Ano-08 / Edição - 138\n"

	if got := (Set{}).For(domain.SourceDiarioCamara).EditionNumber(text); got != "138" {
		t.Errorf("Câmara: %q", got)
	}
	if got := (Set{}).For("").EditionNumber(text); got != "" {
		t.Errorf("fonte vazia deveria ser a Prefeitura: %q", got)
	}
}

func TestCamaraSigningLineAtTheTopOfAPageIsKept(t *testing.T) {
	text := camaraHeader + "PORTARIA Nº 156/2025\nEXONERAR o servidor do gabinete da presidência.\nPágina 1 de 2\n\f" +
		camaraHeader + "São Gonçalo, 30 de outubro de 2025.\nPIERO CABRAL\n"

	acts := ForSource(domain.SourceDiarioCamara).Parse(text)

	if len(acts) != 1 || !strings.Contains(acts[0].Body, "São Gonçalo, 30 de outubro de 2025.\nPIERO CABRAL") {
		t.Fatalf("a assinatura no topo da página sumiu: %+v", acts)
	}
}

func TestCamaraActsHaveNoOrgan(t *testing.T) {
	body := "PORTARIA Nº 1/2025\nNOMEAR o servidor para a comissão.\nSEMAD\nPORTARIA Nº 10/2026\nNomeia servidor para a função.\n"
	if got := organs(body); !strings.Contains(got, "=SEMAD") {
		t.Fatalf("o texto deveria ter órgão para a Prefeitura: %s", got)
	}

	acts := ForSource(domain.SourceDiarioCamara).Parse(camaraHeader + body)

	for _, a := range acts {
		if a.Organ != "" {
			t.Errorf("ato da Câmara com órgão %q: %s", a.Organ, a.Title)
		}
	}
}
