package usecase

import (
	"context"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeSpending struct {
	fiscalCalls int
	pncpCalls   int
}

func (f *fakeSpending) QueryPayments(context.Context, domain.PaymentQuery) (domain.PaymentReport, error) {
	return domain.PaymentReport{ByYear: []domain.SpendingYear{{Year: 2025}}}, nil
}

func (f *fakeSpending) FiscalTotals(context.Context) ([]domain.FiscalTotal, error) {
	f.fiscalCalls++
	return []domain.FiscalTotal{{Year: 2025, Period: domain.LastRREOPeriod, PaidCents: 700}}, nil
}

func (f *fakeSpending) PaidByYear(context.Context) ([]domain.YearPaid, error) { return nil, nil }

func (f *fakeSpending) QueryProcurements(context.Context, domain.ProcurementQuery) (domain.ProcurementReport, error) {
	return domain.ProcurementReport{ProcurementsTotal: 1}, nil
}

func (f *fakeSpending) PNCPContractsBySupplier(context.Context, string) ([]domain.PNCPContract, error) {
	f.pncpCalls++
	return []domain.PNCPContract{{ControlNumber: "x"}}, nil
}

func TestQueryPaymentsAddsFiscalControlOnlyForWholeYears(t *testing.T) {
	f := &fakeSpending{}
	uc := NewQueryPayments(f, f)
	years, _ := domain.NewPaymentQuery("", "", "", "", 2025, 2025, 0)
	entity, _ := domain.NewPaymentQuery("", "saude", "", "", 2025, 2025, 0)

	whole, err := uc.Execute(context.Background(), years)
	if err != nil {
		t.Fatal(err)
	}
	filtered, err := uc.Execute(context.Background(), entity)
	if err != nil {
		t.Fatal(err)
	}

	if whole.ByYear[0].ControlPaidCents == nil || *whole.ByYear[0].ControlPaidCents != 700 {
		t.Fatalf("%+v", whole.ByYear)
	}
	if filtered.ByYear[0].ControlPaidCents != nil || f.fiscalCalls != 1 {
		t.Fatalf("%+v %d", filtered.ByYear, f.fiscalCalls)
	}
}

func TestQueryProcurementsAddsPNCPOnlyForCNPJ(t *testing.T) {
	f := &fakeSpending{}
	uc := NewQueryProcurements(f, f)
	byCNPJ, _ := domain.NewProcurementQuery("12.345.678/0001-95", "", "", "", 0, 0, 0)
	byText, _ := domain.NewProcurementQuery("", "", "merenda", "", 0, 0, 0)

	withPNCP, _ := uc.Execute(context.Background(), byCNPJ)
	without, _ := uc.Execute(context.Background(), byText)

	if len(withPNCP.PNCP) != 1 || len(without.PNCP) != 0 || f.pncpCalls != 1 {
		t.Fatalf("%+v %+v %d", withPNCP.PNCP, without.PNCP, f.pncpCalls)
	}
}
