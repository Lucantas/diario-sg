package pmsg

import "testing"

func TestParseListing(t *testing.T) {
	s, _ := New("https://do.pmsg.rj.gov.br/")
	html := `<ul>
	  <li><a href="/arquivos/2026-09-18.pdf">Edição nº 1234 - 18/09/2026</a></li>
	  <li><a href="https://outro/x.pdf">Sem data</a></li>
	</ul>`
	got := s.parseListing(html)
	if len(got) != 1 {
		t.Fatalf("esperava 1 edição, veio %d", len(got))
	}
	if got[0].Number != "1234" || got[0].URL != "https://do.pmsg.rj.gov.br/arquivos/2026-09-18.pdf" {
		t.Errorf("edição inesperada: %+v", got[0])
	}
}
