package http

import (
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type pncpContractDTO struct {
	ControlNumber string  `json:"control_number"`
	URL           string  `json:"url"`
	OrgCNPJ       string  `json:"org_cnpj"`
	Unit          string  `json:"unit"`
	Kind          string  `json:"kind"`
	Number        string  `json:"number"`
	Process       string  `json:"process"`
	Object        string  `json:"object"`
	ValueCents    int64   `json:"value_cents"`
	SignedAt      *string `json:"signed_at"`
	StartsAt      *string `json:"starts_at"`
	EndsAt        *string `json:"ends_at"`
}

func toPNCPDTOs(contracts []domain.PNCPContract) []pncpContractDTO {
	out := make([]pncpContractDTO, len(contracts))
	for i, c := range contracts {
		out[i] = pncpContractDTO{ControlNumber: c.ControlNumber, URL: c.URL(), OrgCNPJ: c.OrgCNPJ, Unit: c.UnitName, Kind: c.Kind,
			Number: c.Number, Process: c.Process, Object: c.Object, ValueCents: c.ValueCents,
			SignedAt: dayOf(c.SignedAt), StartsAt: dayOf(c.StartsAt), EndsAt: dayOf(c.EndsAt)}
	}
	return out
}

func dayOf(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.DateOnly)
	return &s
}
