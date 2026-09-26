package mcp

import (
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type pncpContractDTO struct {
	Link          string `json:"link"`
	Unidade       string `json:"unidade"`
	Numero        string `json:"numero"`
	Processo      string `json:"processo"`
	Objeto        string `json:"objeto"`
	ValorCentavos int64  `json:"valor_centavos"`
	Assinatura    string `json:"assinatura,omitempty"`
	Vigencia      string `json:"vigencia,omitempty"`
}

func pncpContractsOf(contracts []domain.PNCPContract) []pncpContractDTO {
	out := make([]pncpContractDTO, len(contracts))
	for i, c := range contracts {
		out[i] = pncpContractDTO{Link: c.URL(), Unidade: c.UnitName, Numero: c.Number, Processo: c.Process, Objeto: c.Object,
			ValorCentavos: c.ValueCents, Assinatura: pncpDay(c.SignedAt), Vigencia: periodOf(c.StartsAt, c.EndsAt)}
	}
	return out
}

func pncpDay(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(brDate)
}

func periodOf(from, to *time.Time) string {
	if from == nil && to == nil {
		return ""
	}
	return pncpDay(from) + " a " + pncpDay(to)
}
