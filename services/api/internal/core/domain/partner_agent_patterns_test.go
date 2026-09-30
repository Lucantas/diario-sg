package domain

import (
	"strings"
	"testing"
)

func TestDistinctiveNameNeedsThreeWordsBesidesParticles(t *testing.T) {
	for name, want := range map[string]bool{
		"MARIANA BARBOSA DE SOUZA": true,
		"José Carlos Pereira":      true,
		"MARIA DA SILVA":           false,
		"ANA DE E DOS SANTOS":      false,
		"JOAO SOUZA":               false,
	} {
		if got := IsDistinctiveName(name); got != want {
			t.Errorf("%q: veio %v", name, got)
		}
	}
}

func TestPartnerPublicAgentsMatchPayrollAndAppointmentsByFullName(t *testing.T) {
	person := func(name string) PartnerKey {
		return PartnerKey{Kind: PartnerPerson, Name: name, Document: "***123456**"}
	}
	profiles := map[string]SupplierProfile{
		"11111111000111": {CNPJ: "11111111000111", Name: "OBRAS LTDA", Partners: []PartnerKey{person("JOÃO CARLOS PEREIRA"), person("MARIA DA SILVA")}},
		"11111111000200": {CNPJ: "11111111000200", Name: "OBRAS LTDA", Partners: []PartnerKey{person("JOÃO CARLOS PEREIRA")}},
		"22222222000122": {CNPJ: "22222222000122", Name: "MICAL INVEST LTDA", Partners: []PartnerKey{person("MARIANA BARBOSA DE SOUZA")}},
		"33333333000133": {CNPJ: "33333333000133", Name: "HOLDING SA", Partners: []PartnerKey{{Kind: PartnerCompany, Name: "JOAO CARLOS PEREIRA PARTICIPACOES"}}},
		"44444444000144": {CNPJ: "44444444000144", Name: "SEM CONTRATO LTDA", Partners: []PartnerKey{person("JOÃO CARLOS PEREIRA")}},
	}
	agents := []PublicAgentName{{Name: "Joao Carlos Pereira", Role: RoleSecretario, Office: "SEMOBI"}, {Name: "MARIA DA SILVA", Role: RoleVereador}}
	appointments := []PartnerAppointment{
		{Name: "MARIANA BARBOSA DE SOUZA", ActID: "nomeacao-1"},
		{Name: "MARIANA BARBOSA DE SOUZA", ActID: "nomeacao-2"},
		{Name: "MARIA DA SILVA", ActID: "homonimo"},
	}

	got := FindPartnerPublicAgents([]string{"11111111000111", "11111111000200", "22222222000122", "33333333000133"}, profiles, agents, appointments)

	if len(got) != 2 {
		t.Fatalf("esperava 2 achados: %+v", got)
	}
	if got[0].Profile.CNPJ != "11111111000111" || got[0].Agent == nil || got[0].Agent.Role != RoleSecretario || len(got[0].AppointmentActIDs) != 0 {
		t.Errorf("agente da folha: %+v", got[0])
	}
	if got[1].Profile.CNPJ != "22222222000122" || got[1].Agent != nil || strings.Join(got[1].AppointmentActIDs, ",") != "nomeacao-1,nomeacao-2" {
		t.Errorf("nome em nomeação: %+v", got[1])
	}
}

func TestPartnerPublicAgentFindingsSayTheLinkIsOnlyPossible(t *testing.T) {
	agent := PartnerPublicAgent{Profile: SupplierProfile{CNPJ: "11111111000111", Name: "OBRAS LTDA"},
		Agent: &PublicAgentName{Name: "JOAO CARLOS PEREIRA", Role: RoleSecretario, Office: "SEMOBI"}}
	appointed := PartnerPublicAgent{Profile: SupplierProfile{CNPJ: "22222222000122", Name: "MICAL INVEST LTDA"},
		AppointmentActIDs: []string{"a", "b"}, Appointments: 7}

	fa, fb := PartnerPublicAgentFinding(agent), PartnerPublicAgentFinding(appointed)

	if fa.Title != "OBRAS LTDA (11.111.111/0001-11): sócio com o nome de JOAO CARLOS PEREIRA, Secretário municipal (SEMOBI)" {
		t.Errorf("título do agente: %q", fa.Title)
	}
	if fb.Title != "MICAL INVEST LTDA (22.222.222/0001-22): sócio com o nome de pessoa nomeada ou exonerada no Diário" ||
		strings.Contains(fb.Title+fb.Detail, "MARIANA") {
		t.Errorf("título de nomeação não pode nomear a pessoa: %q %q", fb.Title, fb.Detail)
	}
	if !strings.Contains(fb.Detail, "7 atos de nomeação ou exoneração") || !strings.Contains(fa.Detail, "Ligação possível") ||
		len(fb.ActIDs) != 2 || !fb.Mentions(EntityCNPJ, "22222222000122") {
		t.Errorf("detalhe: %+v %+v", fa, fb)
	}
	if _, ok := PatternCatalog()[PatternPartnerPublicAgent]; !ok {
		t.Error("padrão fora do catálogo")
	}
}
