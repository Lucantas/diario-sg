package http

import (
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type sanctionDTO struct {
	Register    string  `json:"register"`
	Code        string  `json:"code"`
	CNPJ        string  `json:"cnpj"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	StartsAt    *string `json:"starts_at"`
	EndsAt      *string `json:"ends_at"`
	PublishedAt *string `json:"published_at"`
	Process     string  `json:"process"`
	Organ       string  `json:"organ"`
	OrganUF     string  `json:"organ_uf"`
	Sphere      string  `json:"sphere"`
	Scope       string  `json:"scope"`
	LegalBasis  string  `json:"legal_basis"`
	FineCents   *int64  `json:"fine_cents"`
	FirstSeen   string  `json:"first_seen"`
	LastSeen    string  `json:"last_seen"`
	State       string  `json:"state"`
}

func toSanctionDTOs(sanctions []domain.Sanction, listedOn map[string]time.Time, today time.Time) []sanctionDTO {
	out := make([]sanctionDTO, len(sanctions))
	for i, s := range sanctions {
		out[i] = sanctionDTO{Register: s.Register, Code: s.Code, CNPJ: s.CNPJ, Name: s.Name, Category: s.Category,
			StartsAt: optionalDate(s.StartsAt), EndsAt: optionalDate(s.EndsAt), PublishedAt: optionalDate(s.PublishedAt),
			Process: s.Process, Organ: s.Organ, OrganUF: s.OrganUF, Sphere: s.Sphere, Scope: s.Scope, LegalBasis: s.LegalBasis,
			FineCents: s.FineCents, FirstSeen: s.FirstSeen.Format(time.DateOnly), LastSeen: s.LastSeen.Format(time.DateOnly),
			State: string(s.State(listedOn[s.Register], today))}
	}
	return out
}

func listedOnDTO(listedOn map[string]time.Time) map[string]string {
	out := make(map[string]string, len(listedOn))
	for register, day := range listedOn {
		out[register] = day.Format(time.DateOnly)
	}
	return out
}
