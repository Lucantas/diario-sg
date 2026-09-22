package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func pageSpans(acts []domain.Act) []string {
	out := make([]string, 0, len(acts))
	for _, a := range acts {
		out = append(out, fmt.Sprintf("%s %d-%d", a.Title, a.PageStart, a.PageEnd))
	}
	return out
}

func TestParse_PagesFollowFormFeeds(t *testing.T) {
	text := "DECRETO Nº 1/2026\nDispõe sobre o horário.\n\fcontinua o decreto 1.\n" +
		"DECRETO Nº 2/2026\nDispõe sobre a limpeza.\n\fDECRETO Nº 3/2026\nDispõe sobre a feira."

	got := pageSpans(New().Parse(text))

	want := []string{"DECRETO Nº 1/2026 1-2", "DECRETO Nº 2/2026 2-2", "DECRETO Nº 3/2026 3-3"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("esperava %v, veio %v", want, got)
	}
}

func TestParse_TextWithoutFormFeedIsPageOne(t *testing.T) {
	got := pageSpans(New().Parse("DECRETO Nº 1/2026\nDispõe sobre o horário."))

	if len(got) != 1 || got[0] != "DECRETO Nº 1/2026 1-1" {
		t.Errorf("veio %v", got)
	}
}

func TestParse_SplitHeaderAcrossPagesKeepsTheFirstPage(t *testing.T) {
	text := "TERMO\nDE\n\fAPREENSÃO\nADMINISTRATIVA\nNº\n5/2026\nApreendido o veículo de placa ABC1D23."

	got := pageSpans(New().Parse(text))

	want := "TERMO DE APREENSÃO ADMINISTRATIVA Nº 5/2026 1-2"
	if len(got) != 1 || got[0] != want {
		t.Errorf("esperava [%s], veio %v", want, got)
	}
}

func TestParse_DanglingHeaderAcrossPagesIsJoined(t *testing.T) {
	text := "EXTRATO DO QUINTO TERMO ADITIVO DE PRORROGAÇÃO AO\n\fCONTRATO DE LOCAÇÃO 006/2020.\nObjeto: prorrogação do prazo."

	got := pageSpans(New().Parse(text))

	want := "EXTRATO DO QUINTO TERMO ADITIVO DE PRORROGAÇÃO AO CONTRATO DE LOCAÇÃO 006/2020. 1-2"
	if len(got) != 1 || got[0] != want {
		t.Errorf("esperava [%s], veio %v", want, got)
	}
}

func TestRealEditions_ActPagesMatchTheText(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "..", "testdata", "editions", "*.txt"))
	if len(files) == 0 {
		t.Skip("sem edições reais em testdata/editions (rode ./scripts/fetch-editions.sh)")
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		pages := strings.Split(string(raw), "\f")
		onPage := make([]map[string]bool, len(pages))
		anywhere := map[string]bool{}
		for i, p := range pages {
			onPage[i] = map[string]bool{}
			for _, l := range strings.Split(p, "\n") {
				l = strings.TrimSpace(l)
				onPage[i][l] = true
				anywhere[l] = true
			}
		}
		name := filepath.Base(f)
		for _, a := range New().Parse(string(raw)) {
			if a.PageStart < 1 || a.PageEnd < a.PageStart || a.PageEnd > len(pages) {
				t.Errorf("%s: %q com páginas inválidas %d-%d (edição tem %d)", name, a.Title, a.PageStart, a.PageEnd, len(pages))
				continue
			}
			bodyLines := strings.Split(a.Body, "\n")
			first := strings.TrimSpace(bodyLines[0])
			last := strings.TrimSpace(bodyLines[len(bodyLines)-1])
			if anywhere[first] && !onPage[a.PageStart-1][first] {
				t.Errorf("%s: %q começa em %q, que não está na página %d", name, a.Title, first, a.PageStart)
			}
			if anywhere[last] && !onPage[a.PageEnd-1][last] {
				t.Errorf("%s: %q termina em %q, que não está na página %d", name, a.Title, last, a.PageEnd)
			}
		}
	}
}
