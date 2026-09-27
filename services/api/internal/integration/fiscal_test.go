//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestFiscalControlComparesTCEPaymentsWithTheRREO(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)
	ctx := context.Background()
	repo := postgres.NewFiscalRepo(db)
	if err := repo.SaveFiscalTotals(ctx, []domain.FiscalTotal{
		{Year: 2024, Period: 6, CommittedCents: 2000, LiquidatedCents: 1900, PaidCents: 1000, SourceURL: "https://siconfi/2024"},
		{Year: 2025, Period: 5, CommittedCents: 900, LiquidatedCents: 800, PaidCents: 700, SourceURL: "https://siconfi/2025/5"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveFiscalTotals(ctx, []domain.FiscalTotal{{Year: 2025, Period: 6, CommittedCents: 1000, LiquidatedCents: 1000,
		PaidCents: 1000, SourceURL: "https://siconfi/2025/6"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO payments (source, year, month, unit, commitment, cnpj, function, committed_cents, liquidated_cents, paid_cents)
		VALUES ('tce_empenhos', 2024, 1, 'u', '1', '', '', 1000, 900, 500), ('tce_empenhos', 2024, 2, 'u', '2', '', '', 900, 900, 450),
		       ('tce_empenhos', 2025, 1, 'u', '3', '', '', 990, 990, 980)`); err != nil {
		t.Fatal(err)
	}

	var o struct {
		Fiscal []struct {
			Year         int    `json:"year"`
			Period       int    `json:"period"`
			PaidCents    int64  `json:"paid_cents"`
			TCEPaidCents int64  `json:"tce_paid_cents"`
			CoverageBP   int    `json:"paid_coverage_bp"`
			Low          bool   `json:"low_coverage"`
			SourceURL    string `json:"source_url"`
		} `json:"fiscal_control"`
	}
	getJSON(t, srv.URL+"/v1/tce", &o)

	if len(o.Fiscal) != 2 {
		t.Fatalf("controle: %+v", o.Fiscal)
	}
	if f := o.Fiscal[0]; f.Year != 2024 || f.TCEPaidCents != 950 || f.CoverageBP != 9500 || f.Low {
		t.Errorf("2024: %+v", f)
	}
	if f := o.Fiscal[1]; f.Period != 6 || f.TCEPaidCents != 980 || f.CoverageBP != 9800 || f.SourceURL != "https://siconfi/2025/6" {
		t.Errorf("2025 regravado: %+v", f)
	}
}
