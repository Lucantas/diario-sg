package domain

import (
	"strings"
	"testing"
)

func TestEntityWarnings(t *testing.T) {
	many := EntityReport{Kind: EntityProcesso, Certainty: CertaintyStrong, Sources: 1,
		Organs: []OrganCount{{"SEMAD", 3}, {"SEMTRAN", 2}, {"", 1}}}
	if w := EntityWarnings(many); len(w) != 1 || !strings.Contains(w[0], "aparece em 2 órgãos") {
		t.Fatalf("vários órgãos: %v", w)
	}
	weak := EntityReport{Kind: EntityContrato, Certainty: CertaintyWeak, Sources: 1, Organs: []OrganCount{{"FMS", 1}}}
	if w := EntityWarnings(weak); len(w) != 1 || !strings.Contains(w[0], "sem a sigla do órgão") {
		t.Fatalf("certeza fraca: %v", w)
	}
	both := EntityReport{Kind: EntityProcesso, Certainty: CertaintyWeak, Sources: 2}
	if w := EntityWarnings(both); len(w) != 1 || !strings.Contains(w[0], "no Diário da Prefeitura e no da Câmara") {
		t.Fatalf("dois Diários: %v", w)
	}
	if w := EntityWarnings(EntityReport{Kind: EntityCNPJ, Certainty: CertaintyExact, Organs: []OrganCount{{"A", 1}, {"B", 1}}}); len(w) != 0 {
		t.Fatalf("CNPJ em vários órgãos é normal: %v", w)
	}
}
