package mcp

import (
	"fmt"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type paymentYearDTO struct {
	Ano               int      `json:"ano"`
	Unidades          []string `json:"unidades"`
	EmpenhadoCentavos int64    `json:"empenhado_centavos"`
	LiquidadoCentavos int64    `json:"liquidado_centavos"`
	PagoCentavos      int64    `json:"pago_centavos"`
}

func paymentsOf(kind domain.EntityKind, years []domain.PaymentYear) []paymentYearDTO {
	if kind != domain.EntityCNPJ {
		return nil
	}
	out := make([]paymentYearDTO, len(years))
	for i, y := range years {
		out[i] = paymentYearDTO{Ano: y.Year, Unidades: y.Units, EmpenhadoCentavos: y.CommittedCents, LiquidadoCentavos: y.LiquidatedCents, PagoCentavos: y.PaidCents}
	}
	return out
}

func paymentsCoverageOf(kind domain.EntityKind, c *domain.PaymentCoverage) string {
	if kind != domain.EntityCNPJ || c == nil {
		return ""
	}
	return fmt.Sprintf("%02d/%04d a %02d/%04d", c.FromMonth, c.FromYear, c.ToMonth, c.ToYear)
}
