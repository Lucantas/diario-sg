package domain

import "testing"

func TestPhaseOf(t *testing.T) {
	cases := []struct {
		typ   ActType
		title string
		want  Phase
	}{
		{ActContrato, "EXTRATO DE DISTRATO DE CONTRATO", PhaseRescisao},
		{ActOutro, "TERMO DE RESCISÃO UNILATERAL", PhaseRescisao},
		{ActAditivo, "EXTRATO DO SEGUNDO TERMO ADITIVO AO CONTRATO", PhaseAditivo},
		{ActContrato, "EXTRATO DE TERMO ADITVO", PhaseAditivo},
		{ActAditivo, "EXTRATO DO PRIMEIRO TERMO DE APOSTILAMENTO (RERATIFICAÇÃO) DO CONTRATO 009/", PhaseAditivo},
		{ActContrato, "EXTRATO DE AJUSTE DE CONTAS E RECONHECIMENTO DE DÍVIDA", PhaseAjusteContas},
		{ActContrato, "EXTRATO DE NOMEAÇÃO DE FISCAIS", PhaseFiscal},
		{ActOutro, "SUBSTITUIÇÃO DE FISCAL DO CONTRATO Nº 17/2021", PhaseFiscal},
		{ActLicitacao, "HOMOLOGAÇÃO/ADJUDICAÇÃO - CONVITE Nº 001/2011", PhaseHomologacao},
		{ActLicitacao, "EXTRATO DA ATA DE REGISTRO DE PREÇOS Nº 12/2023", PhaseAtaRegistroPrecos},
		{ActContrato, "EXTRATO DE ATA DE REGISTRO DE PREÇO", PhaseAtaRegistroPrecos},
		{ActDispensa, "EXTRATO DE RATIFICAÇÃO", PhaseDispensa},
		{ActOutro, "TERMO DE RATIFICAÇÃO", PhaseDispensa},
		{ActLicitacao, "AVISO DE DISPENSA ELETRÔNICA Nº 90003/2025", PhaseDispensa},
		{ActContrato, "EXTRATO DE CONTRATO", PhaseContrato},
		{ActLicitacao, "AVISO DE LICITAÇÃO", PhaseLicitacao},
		{ActEdital, "EDITAL DE PREGÃO ELETRÔNICO Nº 5/2024", PhaseLicitacao},
		{ActAta, "ATA DA SESSÃO PÚBLICA", PhaseLicitacao},
		{ActDespacho, "DESPACHO DO SECRETÁRIO", PhaseOutro},
		{ActPortaria, "PORTARIA Nº 14/FMS/2022.", PhaseOutro},
		{ActOutro, "AUTO DE INFRAÇÃO Nº 12", PhaseOutro},
	}
	for _, c := range cases {
		if got := PhaseOf(c.typ, c.title); got != c.want {
			t.Errorf("%s %q: esperava %s, veio %s", c.typ, c.title, c.want, got)
		}
	}
}

func TestPhasesListsEveryPhaseOnce(t *testing.T) {
	seen := map[Phase]bool{}
	for _, p := range Phases {
		if seen[p] {
			t.Fatalf("%s repetida", p)
		}
		seen[p] = true
	}
	if len(Phases) != 10 || Phases[0] != PhaseLicitacao || Phases[len(Phases)-1] != PhaseOutro {
		t.Fatalf("ordem inesperada: %v", Phases)
	}
}
