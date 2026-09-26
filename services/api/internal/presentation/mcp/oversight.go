package mcp

import "github.com/seu-usuario/diario-sg/services/api/internal/core/domain"

type stalledWorkDTO struct {
	Contrato           string `json:"contrato"`
	Orgao              string `json:"orgao"`
	ValorCentavos      int64  `json:"valor_contrato_centavos"`
	PagoCentavos       int64  `json:"valor_pago_centavos"`
	ParalisadaEm       string `json:"paralisada_em,omitempty"`
	Motivo             string `json:"motivo"`
	SituacaoDoContrato string `json:"situacao_do_contrato"`
}

func stalledWorksOf(works []domain.StalledWork) []stalledWorkDTO {
	out := make([]stalledWorkDTO, len(works))
	for i, w := range works {
		out[i] = stalledWorkDTO{Contrato: w.Contract, Orgao: w.Organ, ValorCentavos: w.TotalCents, PagoCentavos: w.PaidCents,
			ParalisadaEm: pncpDay(w.StalledAt), Motivo: w.Reason, SituacaoDoContrato: w.ContractStatus}
	}
	return out
}
