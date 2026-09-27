//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

func TestQueryPaymentsFiltersAndTotalsPortalCommitments(t *testing.T) {
	_, db := newServerFor(t, gazetteText)
	ctx := context.Background()
	c := func(entity int, name string, id int64, cnpj, object, process string, paid int64) domain.MunicipalCommitment {
		return domain.MunicipalCommitment{EntityID: entity, Entity: name, Year: 2025, CommitmentID: id, Number: "1",
			Date: time.Date(2025, 5, int(id), 0, 0, 0, 0, time.UTC), CNPJ: cnpj, Name: "CREDOR " + cnpj[:2], Object: object, Process: process,
			CommittedCents: paid, LiquidatedCents: paid, PaidCents: paid}
	}
	if err := postgres.NewMunicipalCommitmentRepo(db).ReplaceMunicipalYear(ctx, 2025, []domain.MunicipalCommitment{
		c(1, "PREFEITURA MUNICIPAL DE SÃO GONÇALO", 1, fpVieira, "merenda escolar", "2888/2024", 100),
		c(18, "FUNDO MUNICIPAL DE SAUDE", 2, fpVieira, "remédios", "02.888/24", 40),
		c(18, "FUNDO MUNICIPAL DE SAUDE", 3, "11111111000191", "Manutenção predial", "5/2025", 7),
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := postgres.NewFiscalRepo(db).SaveFiscalTotals(ctx, []domain.FiscalTotal{{Year: 2025, Period: 6, PaidCents: 500, SourceURL: "u"}}); err != nil {
		t.Fatal(err)
	}
	uc := usecase.NewQueryPayments(postgres.NewMunicipalCommitmentRepo(db), postgres.NewFiscalRepo(db))
	run := func(cnpj, entity, process, text string, from int) domain.PaymentReport {
		t.Helper()
		q, err := domain.NewPaymentQuery(cnpj, entity, process, text, from, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		r, err := uc.Execute(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}

	if r := run("", "saude", "", "", 0); r.Totals.Commitments != 2 || r.Totals.PaidCents != 47 || len(r.ByEntity) != 1 || len(r.BySupplier) != 2 {
		t.Fatalf("por entidade: %+v", r)
	}
	if r := run("", "", "2888/2024", "", 0); r.Totals.Commitments != 2 || r.Totals.PaidCents != 140 {
		t.Fatalf("por processo com grafias diferentes: %+v", r.Totals)
	}
	if r := run("", "", "", "manutencao", 0); r.Totals.Commitments != 1 || r.Commitments[0].CommitmentID != 3 {
		t.Fatalf("texto sem acento: %+v", r)
	}
	if r := run(fpVieira, "", "", "", 0); r.Totals.PaidCents != 140 || r.ByYear[0].ControlPaidCents != nil || r.Commitments[0].CommitmentID != 2 {
		t.Fatalf("por cnpj: %+v", r)
	}
	if r := run("", "", "", "", 2025); r.Totals.PaidCents != 147 || r.ByYear[0].ControlPaidCents == nil || *r.ByYear[0].ControlPaidCents != 500 {
		t.Fatalf("só anos, com controle: %+v", r.ByYear)
	}
	if r := run("", "", "", "", 2026); r.Totals.Commitments != 0 || len(r.Commitments) != 0 {
		t.Fatalf("ano vazio: %+v", r)
	}
}

func TestQueryProcurementsFiltersMuralAndAddsPNCP(t *testing.T) {
	_, db := newServerFor(t, gazetteText)
	ctx := context.Background()
	if err := postgres.NewMunicipalCommitmentRepo(db).ReplaceMunicipalYear(ctx, 2025, []domain.MunicipalCommitment{{EntityID: 1, Entity: "PREFEITURA",
		Year: 2025, CommitmentID: 1, Number: "1", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), CNPJ: fpVieira, Process: "2960/2024"}}, nil); err != nil {
		t.Fatal(err)
	}
	opens := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	if err := postgres.NewProcurementRepo(db).ReplaceMural(ctx, []domain.Procurement{
		{List: domain.MuralTenders, ID: 10, Notice: "PE/1/2025/FMS", Process: "02.960/2024", ProcessKey: "2960/2024", OpensAt: &opens, Object: "Aquisição de remédios"},
		{List: domain.MuralTenders, ID: 11, Notice: "PE/2/2025/PMSG", Process: "9/2025", ProcessKey: "9/2025", Object: "merenda"},
		{List: domain.MuralTenders, ID: 12, Notice: "PE/3/2020/FMSX", Process: "", ProcessKey: "", Object: "outra"},
	}, []domain.ProcurementContract{
		{ProcurementID: 10, Notice: "PE/1/2025/FMS", ProcessKey: "2960/2024", Object: "remédios", Supplier: "F.P. VIEIRA", ValueCents: 100},
		{ProcurementID: 11, Notice: "PE/2/2025/PMSG", ProcessKey: "9/2025", Object: "merenda", Supplier: "OUTRA", ValueCents: 30},
		{ProcurementID: 12, Notice: "PE/3/2020/FMSX", ProcessKey: "", Object: "outra", Supplier: "X", ValueCents: 1},
	}); err != nil {
		t.Fatal(err)
	}
	uc := usecase.NewQueryProcurements(postgres.NewProcurementRepo(db), postgres.NewPNCPRepo(db))
	run := func(cnpj, text, organ string, from int) domain.ProcurementReport {
		t.Helper()
		q, err := domain.NewProcurementQuery(cnpj, "", text, organ, from, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		r, err := uc.Execute(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}

	if r := run(fpVieira, "", "", 0); r.ProcurementsTotal != 1 || r.Procurements[0].ID != 10 || r.ContractsTotal != 1 || r.ContractValueCents != 100 {
		t.Fatalf("por cnpj: %+v", r)
	}
	if r := run("", "", "fms", 0); r.ProcurementsTotal != 1 || r.ContractsTotal != 1 {
		t.Fatalf("por órgão, sem pegar FMSX: %+v", r)
	}
	if r := run("", "remedios", "", 0); r.ProcurementsTotal != 1 || r.ContractsTotal != 1 {
		t.Fatalf("texto sem acento: %+v", r)
	}
	if r := run("", "", "", 2025); r.ProcurementsTotal != 1 || r.Procurements[0].ID != 10 || r.ContractsTotal != 1 || r.Contracts[0].ProcurementID != 11 {
		t.Fatalf("anos (abertura e processo): %+v", r)
	}
	if r := run("11111111000191", "", "", 0); r.ProcurementsTotal != 0 || r.ContractsTotal != 0 {
		t.Fatalf("cnpj sem empenho: %+v", r)
	}
}
