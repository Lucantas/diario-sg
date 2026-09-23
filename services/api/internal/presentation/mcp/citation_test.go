package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

var sample = citable{
	GazetteID: "g1", Title: "PORTARIA Nº 10/2026", EditionNumber: "1771",
	PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
	SourceURL:   "https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf",
	PageStart:   3, PageEnd: 4, Checksum: strings.Repeat("ab", 32),
}

var accessed = time.Date(2026, 9, 22, 15, 0, 0, 0, time.UTC)

func TestCitationMatchesTheSiteFormat(t *testing.T) {
	want := "SÃO GONÇALO (RJ). Diário Oficial do Município de São Gonçalo, ed. 1771, 18 set. 2026, p. 3-4. " +
		"PORTARIA Nº 10/2026. Disponível em: <https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf#page=3>. " +
		"Cópia arquivada em: <https://diario.exemplo/api/v1/gazettes/g1/pdf#page=3>. " +
		"SHA-256 do PDF: " + strings.Repeat("ab", 32) + ". Acesso em: 22 set. 2026."

	if got := formatCitation(sample, "https://diario.exemplo", accessed); got != want {
		t.Fatalf("citação diferente do site:\n%s\n%s", got, want)
	}
}

func TestCitationWithoutNumberPageOrFinalDot(t *testing.T) {
	c := sample
	c.EditionNumber, c.IsExtra, c.PageStart, c.PageEnd, c.Title = "", true, 0, 0, "Nomeia:"

	got := formatCitation(c, "https://diario.exemplo", accessed)

	for _, part := range []string{"ed. s/n (extra), 18 set. 2026. Nomeia. Disponível", "<https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf>"} {
		if !strings.Contains(got, part) {
			t.Errorf("faltou %q em %q", part, got)
		}
	}
	if strings.Contains(got, "#page") {
		t.Errorf("sem página não tem #page: %q", got)
	}
}

func TestSourceOfAnActPointsToThePage(t *testing.T) {
	f := sourceOf(sample, "https://diario.exemplo")

	if f.URL != "https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf#page=3" ||
		f.ArchivedCopy != "https://diario.exemplo/api/v1/gazettes/g1/pdf#page=3" ||
		f.Page != 3 || f.SHA256 != sample.Checksum || f.Name != "Diário Oficial do Município de São Gonçalo, edição 1771 de 18/09/2026" {
		t.Fatalf("fonte inesperada: %+v", f)
	}
}

var camaraSample = citable{
	GazetteID: "g2", Title: "PORTARIA Nº 156/2025", EditionNumber: "138", Source: "diario_camara",
	PublishedAt: time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC),
	SourceURL:   "https://www.cmsg.rj.gov.br/diariooficialeletronico/PUBLICACOES/2025-11-03.pdf",
	PageStart:   1, PageEnd: 1, Checksum: strings.Repeat("cd", 32),
}

func TestCitationOfTheCamara(t *testing.T) {
	want := "SÃO GONÇALO (RJ). Câmara Municipal. Diário Oficial Eletrônico da Câmara Municipal de São Gonçalo, ed. 138, 3 nov. 2025, p. 1. " +
		"PORTARIA Nº 156/2025. Disponível em: <https://www.cmsg.rj.gov.br/diariooficialeletronico/PUBLICACOES/2025-11-03.pdf#page=1>. " +
		"Cópia arquivada em: <https://diario.exemplo/api/v1/gazettes/g2/pdf#page=1>. " +
		"SHA-256 do PDF: " + strings.Repeat("cd", 32) + ". Acesso em: 22 set. 2026."

	if got := formatCitation(camaraSample, "https://diario.exemplo", accessed); got != want {
		t.Fatalf("citação da Câmara:\n%s\n%s", got, want)
	}
	if f := sourceOf(camaraSample, "https://diario.exemplo"); f.Name != "Diário Oficial Eletrônico da Câmara Municipal de São Gonçalo, edição 138 de 03/11/2025" {
		t.Errorf("nome da fonte da Câmara: %q", f.Name)
	}
}

func TestCoverageHasOneEntryPerSourceWithItsGaps(t *testing.T) {
	cov := coveragesOf([]domain.Coverage{
		{Source: domain.SourceDiarioPrefeitura, First: time.Date(2010, 1, 4, 0, 0, 0, 0, time.UTC), Last: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)},
		{Source: domain.SourceDiarioCamara},
	})

	if len(cov) != 2 || cov[0].Source != "diario_prefeitura" || cov[0].From != "2010-01-04" || cov[1].Source != "diario_camara" || cov[1].From != "" {
		t.Fatalf("cobertura inesperada: %+v", cov)
	}
	if strings.Contains(strings.Join(cov[0].Gaps, " "), "2020-10-04") || !strings.Contains(strings.Join(cov[1].Gaps, " "), "2020-10-04") {
		t.Errorf("a lacuna de 2020 é só da Câmara: %+v", cov)
	}
	for _, c := range cov {
		if strings.Contains(strings.Join(c.Gaps, " "), "ainda não é coletado") {
			t.Errorf("lacuna antiga da Câmara ficou: %+v", c.Gaps)
		}
	}
}

func TestFilterOfAcceptsTheSource(t *testing.T) {
	f, err := filterOf(searchInput{Diario: "diario_camara"})
	if err != nil || f.Source != "diario_camara" {
		t.Fatalf("fonte: %+v %v", f, err)
	}
}

func TestPageRange(t *testing.T) {
	for _, c := range []struct {
		start, end int
		want       string
	}{{0, 0, ""}, {3, 3, "3"}, {3, 0, "3"}, {3, 5, "3-5"}} {
		if got := pageRange(c.start, c.end); got != c.want {
			t.Errorf("%d-%d: veio %q", c.start, c.end, got)
		}
	}
}

func TestReaisToCents(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want int64
	}{{0, 0}, {1500.5, 150050}, {0.1 + 0.2, 30}} {
		got, err := reaisToCents(c.in)
		if err != nil || got != c.want {
			t.Errorf("%v: veio %d %v", c.in, got, err)
		}
	}
	if _, err := reaisToCents(-1); err == nil {
		t.Error("valor negativo deveria ser erro")
	}
}
