//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestMuralReachesTheCompanyThroughTheCommitmentProcess(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)
	ctx := context.Background()
	if err := postgres.NewMunicipalCommitmentRepo(db).ReplaceMunicipalYear(ctx, 2025, []domain.MunicipalCommitment{{EntityID: 1, Entity: "PREFEITURA",
		Year: 2025, CommitmentID: 1, Number: "1", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), CNPJ: fpVieira, Process: "2960/2026"}}, nil); err != nil {
		t.Fatal(err)
	}
	opens := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	procurements := []domain.Procurement{
		{List: domain.MuralTenders, ID: 10, Notice: "PE/1/2026", Process: "02.960/2026", ProcessKey: "2960/2026", OpensAt: &opens, URL: "https://mural/l?10"},
		{List: domain.MuralTenders, ID: 11, Notice: "PE/2/2026", Process: "9/2026", ProcessKey: "9/2026", URL: "https://mural/l?11"},
	}
	contracts := []domain.ProcurementContract{{ProcurementID: 10, Process: "02.960/2026", ProcessKey: "2960/2026", Supplier: "F.P. VIEIRA",
		ValueCents: 100, Instrument: "Contrato", DocumentURL: "https://mural/d?1"}}
	repo := postgres.NewProcurementRepo(db)
	if err := repo.ReplaceMural(ctx, procurements, contracts); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceMural(ctx, procurements, contracts); err != nil {
		t.Fatal(err)
	}

	var company struct {
		Mural struct {
			Procurements []struct {
				ID      int     `json:"id"`
				OpensAt *string `json:"opens_at"`
				URL     string  `json:"url"`
			} `json:"procurements"`
			Contracts []struct {
				Supplier    string `json:"supplier"`
				DocumentURL string `json:"document_url"`
			} `json:"contracts"`
		} `json:"procurements"`
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/"+fpVieira, &company)

	m := company.Mural
	if len(m.Procurements) != 1 || m.Procurements[0].ID != 10 || m.Procurements[0].OpensAt == nil || *m.Procurements[0].OpensAt != "2026-03-01" ||
		len(m.Contracts) != 1 || m.Contracts[0].DocumentURL != "https://mural/d?1" {
		t.Fatalf("mural da empresa: %+v", m)
	}
}
