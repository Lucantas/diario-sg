package domain

import (
	"errors"
	"testing"
	"time"
)

func billWith(status string, lawNumber int, texts ...string) Bill {
	b := Bill{Status: status, LawNumber: lawNumber}
	at := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	for i, text := range texts {
		b.Events = append(b.Events, BillEvent{Position: i + 1, At: at.AddDate(0, 0, -i), Text: text})
	}
	return b
}

func TestPhaseOfRealTimelines(t *testing.T) {
	cases := []struct {
		name string
		bill Bill
		want BillPhase
	}{
		{"virou lei e foi arquivado no fim do mandato", billWith("Arquivado", 1147, "Arquivado término de mandato",
			"Lei nº. 1147/2020 de 05/02/2020 Publicada em 06/02/2020", "Enviado para Prefeitura Municipal de São Gonçalo - Ofício nº. 1063/2020 em 09/01/2020"), PhaseLaw},
		{"lei na tramitação sem o selo", billWith("Ativo", 0, "Lei nº. 1147/2020 de 05/02/2020 Publicada em 06/02/2020"), PhaseLaw},
		{"arquivado por fim de mandato", billWith("Arquivado", 0, "Processo Arquivado - Término de mandato", "Parecer Aprovado definido pelo relator MISAEL"), PhaseArchived},
		{"vetado", billWith("Ativo", 0, "Veto Total mantido em Plenário", "Enviado para PREFEITURA MUNICIPAL DE SÃO GONÇALO - Ofício nº. 12/2024"), PhaseVetoed},
		{"retirado pelo autor", billWith("Ativo", 0, "Retirado pelo autor", "Entrada no Protocolo Geral - Regime de tramitação Ordinário"), PhaseWithdrawn},
		{"rejeitado", billWith("Ativo", 0, "Rejeitado - Votação única por 12 votos na Sessão de 01/02/2024"), PhaseRejected},
		{"enviado ao Executivo", billWith("Ativo", 0, "Enviado para PREFEITURA MUNICIPAL DE SÃO GONÇALO - Ofício nº. 462/2025 em 29/12/2025",
			"Aprovado - Votação única por 22 votos na Sessão de 23/12/2025 às 10:00"), PhaseSentToExecutive},
		{"ofício aguardando envio", billWith("Ativo", 0, "Ofício 462/2025 - Aguardando envio", "Aprovado - Votação única por 22 votos na Sessão de 23/12/2025 às 10:00"), PhaseApproved},
		{"aprovado", billWith("Ativo", 0, "Aprovado - Votação única por 22 votos na Sessão de 23/12/2025 às 10:00", "Encaminhado ao setor Para Votação"), PhaseApproved},
		{"pronto para votação", billWith("Ativo", 0, "Encaminhado ao setor Para Votação", "Parecer Aprovado definido pelo relator ALAN RODRIGUES finalizado na Comissão"), PhaseVoting},
		{"em comissão", billWith("Ativo", 0, "Definida Relatoria - Vereador MISAEL", "Recebido na Comissão", "Encaminhado a Comissão COMISSÃO DE JUSTIÇA E REDAÇÃO"), PhaseCommittee},
		{"parecer aprovado ainda é comissão", billWith("Ativo", 0, "Parecer Aprovado distribuído para assinatura por  ALEXANDRE GOMES!"), PhaseCommittee},
		{"só apresentado", billWith("Ativo", 0, "Lido no Expediente - Sessão de Quarta - feira, 01 de Julho de 2020", "Entrada no Protocolo Geral - Regime de tramitação Ordinário"), PhasePresented},
		{"sem eventos", billWith("Ativo", 0), PhasePresented},
	}
	for _, c := range cases {
		if got := BillPhaseOf(c.bill); got != c.want {
			t.Errorf("%s: veio %s, esperava %s", c.name, got, c.want)
		}
	}
}

func TestDaysIdleCountsFromTheLatestEvent(t *testing.T) {
	b := billWith("Ativo", 0, "Recebido na Comissão", "Entrada no Protocolo Geral")
	today := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)

	if got := DaysIdle(b, today); got != 26 {
		t.Fatalf("dias: %d", got)
	}
	if got := DaysIdle(Bill{}, today); got != 0 {
		t.Fatalf("sem eventos: %d", got)
	}
}

func TestParseBillPhase(t *testing.T) {
	for _, p := range BillPhasesInOrder {
		if got, err := ParseBillPhase(string(p)); err != nil || got != p {
			t.Errorf("%s: %v %v", p, got, err)
		}
	}
	if got, err := ParseBillPhase(""); err != nil || got != "" {
		t.Errorf("vazio: %v %v", got, err)
	}
	if _, err := ParseBillPhase("sancionado"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("fase desconhecida: %v", err)
	}
}

func TestBillReferenceFromNormAuthors(t *testing.T) {
	cases := map[string]BillDocRef{
		"PROJETO DE LEI Nº 0133/2019 VEREADOR ´PROFESSOR JOSEMAR":  {Kind: "PROJETO DE LEI", Number: 133, Year: 2019},
		"PROJETO D LEI N 118/2019 VEREADOR BRUNO PORTO":            {Kind: "PROJETO DE LEI", Number: 118, Year: 2019},
		"PROJETO DE LEI 0254/2018 VEREADOR PAULO CESAR":            {Kind: "PROJETO DE LEI", Number: 254, Year: 2018},
		"VEREADOR JALMIR JUNIOR PROJETO DE LEI 110/22":             {Kind: "PROJETO DE LEI", Number: 110, Year: 2022},
		"PROJETO DE LEI COMPLEMENTAR Nº 3/2021 PODER EXECUTIVO":    {Kind: "PROJETO DE LEI COMPLEMENTAR", Number: 3, Year: 2021},
		"PROJETO DE LEI Nº 0135/2019 - VEREADOR PROFESSOR JOSEMAR": {Kind: "PROJETO DE LEI", Number: 135, Year: 2019},
	}
	for author, want := range cases {
		got, ok := BillReference(author)
		if !ok || got != want {
			t.Errorf("%q: veio %+v %v", author, got, ok)
		}
	}
	for _, author := range []string{"VER. DINEY MARINS", "", "PREFEITURA MUNICIPAL"} {
		if _, ok := BillReference(author); ok {
			t.Errorf("%q não cita projeto", author)
		}
	}
}
