package domain

import (
	"errors"
	"os"
	"testing"
	"time"
)

const sicamBase = "https://sg.processolegislativo.com.br"

func readSICAM(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/sicam/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func parseSICAM(t *testing.T, name string) Bill {
	t.Helper()
	b, err := ParseBillPage(readSICAM(t, name), sicamBase)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseBillPageMessageSentToExecutive(t *testing.T) {
	b := parseSICAM(t, "5564-2025.html")

	if b.Key != (BillKey{5564, 2025}) || b.Kind != "MENSAGEM" || b.DocLabel != "MENSAGEM Nº 032/2025" || b.DocNumber != 32 || b.DocYear != 2025 {
		t.Fatalf("cabeçalho: %+v", b)
	}
	if b.Status != "Ativo" || b.Summary != "ALTERA A LEI Nº 148 DE 08 DE JULHO DE 2008" || b.Authors != "PREFEITURA MUNICIPAL DE SÃO GONÇALO" {
		t.Fatalf("situação, ementa ou autor: %q %q %q", b.Status, b.Summary, b.Authors)
	}
	if b.PresentedOn == nil || b.PresentedOn.Format(time.DateOnly) != "2025-12-18" {
		t.Fatalf("apresentação: %v", b.PresentedOn)
	}
	if b.CurrentBody != "Câmara Municipal" || b.LastMovement != "Enviado para PREFEITURA MUNICIPAL DE SÃO GONÇALO - Ofício nº. 462/2025 em 29/12/2025" {
		t.Fatalf("órgão ou última movimentação: %q %q", b.CurrentBody, b.LastMovement)
	}
	if b.SourceUpdatedAt == nil || !b.SourceUpdatedAt.Equal(time.Date(2026, 1, 6, 14, 27, 0, 0, time.UTC)) {
		t.Fatalf("última atualização: %v", b.SourceUpdatedAt)
	}
	if b.URL != sicamBase+"/areapublica/processo/5564-2025" || b.LawNumber != 0 {
		t.Fatalf("url ou lei: %q %d", b.URL, b.LawNumber)
	}
	if len(b.Events) != 22 {
		t.Fatalf("eventos: %d", len(b.Events))
	}
	first := b.Events[0]
	if first.Position != 1 || !first.At.Equal(time.Date(2026, 1, 6, 14, 27, 0, 0, time.UTC)) || first.Label != "Movimentado" ||
		first.Text != "Enviado para PREFEITURA MUNICIPAL DE SÃO GONÇALO - Ofício nº. 462/2025 em 29/12/2025" {
		t.Fatalf("primeiro evento: %+v", first)
	}
	if !hasEvent(b, "Encaminhado ao setor Setor de Expediente", "Setor de Expediente") {
		t.Fatalf("evento com setor ausente: %+v", b.Events)
	}
	if len(b.Opinions) != 3 {
		t.Fatalf("pareceres: %+v", b.Opinions)
	}
	cjr := b.Opinions[2]
	if cjr.Result != "Aprovado" || cjr.Committee != "COMISSÃO DE JUSTIÇA E REDAÇÃO" || cjr.Rapporteur != "NELSINHO RUAS" || cjr.On == nil || cjr.On.Format(time.DateOnly) != "2025-12-22" {
		t.Fatalf("parecer da CJR: %+v", cjr)
	}
}

func hasEvent(b Bill, text, sector string) bool {
	for _, e := range b.Events {
		if e.Text == text && e.Sector == sector {
			return true
		}
	}
	return false
}

func TestParseBillPageArchivedAtEndOfTerm(t *testing.T) {
	b := parseSICAM(t, "3865-2019.html")

	if b.Kind != "PROJETO DE LEI" || b.DocLabel != "PROJETO DE LEI Nº 270/2019" || b.DocNumber != 270 || b.Status != "Arquivado" || b.Authors != "PAULO CESAR" {
		t.Fatalf("cabeçalho: %+v", b)
	}
	if len(b.Events) != 9 || b.Events[0].Text != "Processo Arquivado - Término de mandato" || b.Events[8].Text != "Entrada no Protocolo Geral - Regime de tramitação Ordinário" {
		t.Fatalf("eventos: %+v", b.Events)
	}
	if len(b.Opinions) != 1 || b.Opinions[0].Rapporteur != "MISAEL" {
		t.Fatalf("pareceres: %+v", b.Opinions)
	}
}

func TestParseBillPageBillThatBecameLaw(t *testing.T) {
	b := parseSICAM(t, "3823-2019.html")

	if b.LawKind != NormLaw || b.LawNumber != 1147 || b.LawYear != 2020 || b.LawURL != sicamBase+"/areapublica/documento/?Lei/216" {
		t.Fatalf("lei: %s %d/%d %q", b.LawKind, b.LawNumber, b.LawYear, b.LawURL)
	}
	if b.LawLabel() != "Lei nº 1147/2020" {
		t.Fatalf("rótulo: %q", b.LawLabel())
	}
	if b.Status != "Arquivado" || b.DocLabel != "PROJETO DE LEI Nº 268/2019" {
		t.Fatalf("cabeçalho: %+v", b)
	}
	if !hasEvent(b, "Lei nº. 1147/2020 de 05/02/2020 Publicada em 06/02/2020", "") {
		t.Fatalf("evento da lei ausente: %+v", b.Events)
	}
}

func TestParseBillPageResolutionBadgeKeepsTheNormKind(t *testing.T) {
	b := parseSICAM(t, "3997-2026.html")

	if b.Kind != "PROJETO DE RESOLUÇÃO" || b.LawKind != NormResolution || b.LawNumber != 1054 || b.LawYear != 2026 {
		t.Fatalf("resolução: %q %s %d/%d", b.Kind, b.LawKind, b.LawNumber, b.LawYear)
	}
	if b.LawLabel() != "Resolução nº 1054/2026" {
		t.Fatalf("rótulo: %q", b.LawLabel())
	}
}

func TestNormKindFromSICAMBadge(t *testing.T) {
	cases := map[string]NormKind{
		"Lei": NormLaw, "Lei Complementar": NormComplementary, "Resolução": NormResolution,
		"Emenda à Lei Orgânica": NormOrganicAmendment, "Decreto Legislativo": NormLegislativeDecree, "Portaria": "",
	}
	for label, want := range cases {
		if got := NormKindFromSICAM(label); got != want {
			t.Errorf("%q: veio %q, esperava %q", label, got, want)
		}
	}
}

func TestParseBillPageIndication(t *testing.T) {
	b := parseSICAM(t, "100-2025.html")

	if b.Kind != "INDICAÇÃO LEGISLATIVA" || b.DocNumber != 49 || b.Authors != "JULIANO FREITAS" || len(b.Events) == 0 {
		t.Fatalf("indicação: %+v", b)
	}
}

func TestParseBillPageNotFound(t *testing.T) {
	_, err := ParseBillPage(readSICAM(t, "inexistente.html"), sicamBase)

	if !errors.Is(err, ErrBillNotFound) {
		t.Fatalf("esperava ErrBillNotFound, veio %v", err)
	}
}

func TestParseProcessSitemap(t *testing.T) {
	keys, err := ParseProcessSitemap(readSICAM(t, "sitemap-processos.xml"))
	if err != nil {
		t.Fatal(err)
	}

	want := []BillKey{{4431, 2026}, {2484, 2017}, {211, 2014}}
	if len(keys) != len(want) {
		t.Fatalf("chaves: %v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("chaves: %v", keys)
		}
	}
}

func TestParseSitemapIndexListsProcessSitemaps(t *testing.T) {
	urls, err := ParseSitemapIndex(readSICAM(t, "sitemap.xml"), "sitemap-processos-")

	if err != nil || len(urls) != 2 || urls[0] != sicamBase+"/sitemap-processos-1.xml" {
		t.Fatalf("índice: %v %v", urls, err)
	}
}

func TestParseBillKey(t *testing.T) {
	for _, s := range []string{"5564/2025", "5564-2025", "5564_2025", " 5564/2025 "} {
		k, err := ParseBillKey(s)
		if err != nil || k != (BillKey{5564, 2025}) || k.String() != "5564/2025" || k.Slug() != "5564-2025" {
			t.Errorf("%q: %v %v", s, k, err)
		}
	}
	for _, s := range []string{"", "5564", "abc/2025", "0/2025"} {
		if _, err := ParseBillKey(s); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%q deveria ser inválido: %v", s, err)
		}
	}
}

func TestBillPageMainKeepsOnlyMain(t *testing.T) {
	page := []byte(`<html><head><script>x</script></head><body><nav>menu</nav><main id="m">conteúdo</main><footer>f</footer></body></html>`)

	if got := string(BillPageMain(page)); got != `<main id="m">conteúdo</main>` {
		t.Fatalf("main: %q", got)
	}
}
