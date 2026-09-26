package http

import (
	"fmt"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type paymentYearDTO struct {
	Year            int      `json:"year"`
	Units           []string `json:"units"`
	CommittedCents  int64    `json:"committed_cents"`
	LiquidatedCents int64    `json:"liquidated_cents"`
	PaidCents       int64    `json:"paid_cents"`
}

type paymentCoverageDTO struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func toPaymentDTOs(years []domain.PaymentYear) []paymentYearDTO {
	out := make([]paymentYearDTO, len(years))
	for i, y := range years {
		out[i] = paymentYearDTO{Year: y.Year, Units: y.Units, CommittedCents: y.CommittedCents, LiquidatedCents: y.LiquidatedCents, PaidCents: y.PaidCents}
	}
	return out
}

func toPaymentCoverageDTO(c *domain.PaymentCoverage) *paymentCoverageDTO {
	if c == nil {
		return nil
	}
	return &paymentCoverageDTO{From: fmt.Sprintf("%04d-%02d", c.FromYear, c.FromMonth), To: fmt.Sprintf("%04d-%02d", c.ToYear, c.ToMonth)}
}
