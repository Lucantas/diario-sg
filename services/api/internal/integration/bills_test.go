//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func sampleBill(number, year int, kind, status string, fetched time.Time, events ...string) domain.Bill {
	b := domain.Bill{Key: domain.BillKey{Number: number, Year: year}, Kind: kind, DocLabel: kind + " Nº 1/2025", DocNumber: 1, DocYear: year,
		Summary: "DISPÕE SOBRE A ARBORIZAÇÃO URBANA", Authors: "VEREADOR TESTE", Status: status, CurrentBody: "Câmara Municipal",
		URL: "https://sicam/" + domain.BillKey{Number: number, Year: year}.Slug(), FetchedAt: fetched}
	for i, text := range events {
		b.Events = append(b.Events, domain.BillEvent{Position: i + 1, At: fetched.Add(-time.Duration(i) * time.Hour), Label: "Movimentado", Text: text})
	}
	return b
}

func TestSaveBillsReplacesEventsAndListsStaleOpenBills(t *testing.T) {
	_, db := newServerFor(t, gazetteText)
	ctx := context.Background()
	repo := postgres.NewBillRepo(db)
	day := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	first := []domain.Bill{
		sampleBill(10, 2025, "PROJETO DE LEI", "Ativo", day, "Recebido na Comissão", "Entrada no Protocolo Geral"),
		sampleBill(11, 2025, "PROJETO DE LEI", "Arquivado", day, "Processo Arquivado"),
		sampleBill(12, 2025, "INDICAÇÃO LEGISLATIVA", "Ativo", day, "Entrada no Protocolo Geral"),
		sampleBill(13, 2025, "MENSAGEM", "Ativo", day.Add(time.Hour), "Entrada no Protocolo Geral"),
	}
	first[0].Opinions = []domain.BillOpinion{{Position: 1, Result: "Aprovado", Committee: "COMISSÃO DE DEFESA DO MEIO AMBIENTE", Rapporteur: "X"}}
	if err := repo.SaveBills(ctx, first); err != nil {
		t.Fatal(err)
	}
	again := sampleBill(10, 2025, "PROJETO DE LEI", "Ativo", day.Add(48*time.Hour), "Aprovado - Votação única por 20 votos")
	if err := repo.SaveBills(ctx, []domain.Bill{again}); err != nil {
		t.Fatal(err)
	}

	known, err := repo.KnownBills(ctx)
	if err != nil || len(known) != 4 || !known[domain.BillKey{Number: 12, Year: 2025}] {
		t.Fatalf("conhecidos: %v %v", known, err)
	}
	var events, opinions int
	if err := db.QueryRow(`SELECT (SELECT count(*) FROM bill_events WHERE process_number = 10), (SELECT count(*) FROM bill_opinions WHERE process_number = 10)`).Scan(&events, &opinions); err != nil {
		t.Fatal(err)
	}
	if events != 1 || opinions != 0 {
		t.Fatalf("a segunda leitura deveria trocar eventos e pareceres: %d %d", events, opinions)
	}
	stale, err := repo.StaleOpenBills(ctx, domain.NormativeBillKinds, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 2 || stale[0] != (domain.BillKey{Number: 13, Year: 2025}) || stale[1] != (domain.BillKey{Number: 10, Year: 2025}) {
		t.Fatalf("parados abertos, dos lidos há mais tempo para os mais recentes: %v", stale)
	}
	if limited, _ := repo.StaleOpenBills(ctx, domain.NormativeBillKinds, 1); len(limited) != 1 {
		t.Fatalf("limite: %v", limited)
	}
}
