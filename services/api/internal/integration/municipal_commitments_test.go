//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestMunicipalCommitmentsReachTheCompanyAndTheFiscalControl(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)
	ctx := context.Background()
	repo := postgres.NewMunicipalCommitmentRepo(db)
	commitment := func(id int64, day int, paid int64) domain.MunicipalCommitment {
		return domain.MunicipalCommitment{EntityID: 1, Entity: "PREFEITURA", Year: 2025, CommitmentID: id, Number: "14", Date: time.Date(2025, 8, day, 0, 0, 0, 0, time.UTC),
			CNPJ: fpVieira, Name: "F.P. VIEIRA", Object: "obra", ProcessKind: "Licitação", Process: "2888/2024", Modality: "Concorrência 1/2024",
			CommittedCents: 2 * paid, LiquidatedCents: paid, PaidCents: paid}
	}
	if err := repo.ReplaceMunicipalYear(ctx, 2025, []domain.MunicipalCommitment{commitment(1, 1, 999)}, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceMunicipalYear(ctx, 2025, []domain.MunicipalCommitment{commitment(1, 1, 100), commitment(2, 20, 50)},
		[]domain.MunicipalTotal{{Year: 2025, EntityID: 1, Entity: "PREFEITURA", PaidCents: 900}, {Year: 2025, EntityID: 18, Entity: "FMS", PaidCents: 100}}); err != nil {
		t.Fatal(err)
	}
	if err := postgres.NewFiscalRepo(db).SaveFiscalTotals(ctx, []domain.FiscalTotal{{Year: 2025, Period: 6, PaidCents: 1000, SourceURL: "u"}}); err != nil {
		t.Fatal(err)
	}

	var company struct {
		Municipal struct {
			Years []struct {
				Year        int   `json:"year"`
				Commitments int   `json:"commitments"`
				PaidCents   int64 `json:"paid_cents"`
			} `json:"years"`
			Recent []struct {
				Date    string `json:"date"`
				Process string `json:"process"`
			} `json:"recent"`
		} `json:"municipal_commitments"`
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/"+fpVieira, &company)
	if m := company.Municipal; len(m.Years) != 1 || m.Years[0].Commitments != 2 || m.Years[0].PaidCents != 150 ||
		len(m.Recent) != 2 || m.Recent[0].Date != "2025-08-20" || m.Recent[0].Process != "2888/2024" {
		t.Fatalf("empresa: %+v", m)
	}

	var tce struct {
		Fiscal []struct {
			PortalPaidCents *int64 `json:"portal_paid_cents"`
		} `json:"fiscal_control"`
	}
	getJSON(t, srv.URL+"/v1/tce", &tce)
	if len(tce.Fiscal) != 1 || tce.Fiscal[0].PortalPaidCents == nil || *tce.Fiscal[0].PortalPaidCents != 1000 {
		t.Fatalf("controle: %+v", tce.Fiscal)
	}
}
