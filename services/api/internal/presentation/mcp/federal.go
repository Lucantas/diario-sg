package mcp

import "github.com/seu-usuario/diario-sg/services/api/internal/core/domain"

type amendmentPaymentDTO struct {
	Emenda        string `json:"emenda"`
	Autor         string `json:"autor"`
	Mes           string `json:"mes"`
	ValorCentavos int64  `json:"valor_centavos"`
}

func amendmentPaymentsOf(payments []domain.AmendmentPayment) []amendmentPaymentDTO {
	out := make([]amendmentPaymentDTO, len(payments))
	for i, p := range payments {
		out[i] = amendmentPaymentDTO{Emenda: p.Code, Autor: p.Author, Mes: p.Month.Format("01/2006"), ValorCentavos: p.ValueCents}
	}
	return out
}
