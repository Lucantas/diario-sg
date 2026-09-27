package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type QueryPayments struct {
	payments ports.PaymentQueryReader
	fiscal   ports.FiscalReader
}

func NewQueryPayments(payments ports.PaymentQueryReader, fiscal ports.FiscalReader) *QueryPayments {
	return &QueryPayments{payments: payments, fiscal: fiscal}
}

func (uc *QueryPayments) Execute(ctx context.Context, q domain.PaymentQuery) (domain.PaymentReport, error) {
	report, err := uc.payments.QueryPayments(ctx, q)
	if err != nil || !q.OnlyYears() {
		return report, err
	}
	totals, err := uc.fiscal.FiscalTotals(ctx)
	if err != nil {
		return report, err
	}
	report.ByYear = domain.WithFiscalControl(report.ByYear, totals)
	return report, nil
}

type QueryProcurements struct {
	procurements ports.ProcurementQueryReader
	pncp         ports.PNCPReader
}

func NewQueryProcurements(procurements ports.ProcurementQueryReader, pncp ports.PNCPReader) *QueryProcurements {
	return &QueryProcurements{procurements: procurements, pncp: pncp}
}

func (uc *QueryProcurements) Execute(ctx context.Context, q domain.ProcurementQuery) (domain.ProcurementReport, error) {
	report, err := uc.procurements.QueryProcurements(ctx, q)
	if err != nil || q.CNPJ == "" {
		return report, err
	}
	report.PNCP, err = uc.pncp.PNCPContractsBySupplier(ctx, q.CNPJ)
	return report, err
}
