package http

import (
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type activityDTO struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type partnerDTO struct {
	Kind     string  `json:"kind"`
	Name     string  `json:"name"`
	Document string  `json:"document"`
	Role     string  `json:"role"`
	Since    *string `json:"since"`
}

type registryDTO struct {
	Month           string        `json:"month"`
	Name            string        `json:"name"`
	TradeName       string        `json:"trade_name"`
	LegalNature     string        `json:"legal_nature"`
	CapitalCents    int64         `json:"capital_cents"`
	Size            string        `json:"size"`
	Headquarters    bool          `json:"headquarters"`
	Status          string        `json:"status"`
	StatusSince     *string       `json:"status_since"`
	StatusReason    string        `json:"status_reason"`
	OpenedAt        *string       `json:"opened_at"`
	MainActivity    activityDTO   `json:"main_activity"`
	OtherActivities []activityDTO `json:"other_activities"`
	Street          string        `json:"street"`
	Number          string        `json:"number"`
	Complement      string        `json:"complement"`
	District        string        `json:"district"`
	ZIP             string        `json:"zip"`
	City            string        `json:"city"`
	UF              string        `json:"uf"`
	Partners        []partnerDTO  `json:"partners"`
}

var partnerKinds = map[domain.PartnerKind]string{
	domain.PartnerCompany: "pessoa_juridica", domain.PartnerPerson: "pessoa_fisica", domain.PartnerForeign: "estrangeiro",
}

func optionalDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.DateOnly)
	return &s
}

func toRegistryDTO(r *domain.CompanyRegistry) *registryDTO {
	if r == nil {
		return nil
	}
	e := r.Establishment
	out := &registryDTO{Month: r.Month.Format("2006-01"), Name: r.Company.Name, TradeName: e.TradeName,
		LegalNature: r.Company.LegalNature, CapitalCents: r.Company.CapitalCents, Size: r.Company.Size,
		Headquarters: e.Headquarters, Status: e.Status, StatusSince: optionalDate(e.StatusSince), StatusReason: e.StatusReason,
		OpenedAt: optionalDate(e.OpenedAt), MainActivity: activityDTO(e.MainActivity), OtherActivities: make([]activityDTO, len(e.OtherActivities)),
		Street: e.Street, Number: e.Number, Complement: e.Complement, District: e.District, ZIP: e.ZIP, City: e.City, UF: e.UF,
		Partners: make([]partnerDTO, len(r.Partners))}
	for i, a := range e.OtherActivities {
		out.OtherActivities[i] = activityDTO(a)
	}
	for i, p := range r.Partners {
		out.Partners[i] = partnerDTO{Kind: partnerKinds[p.Kind], Name: p.Name, Document: p.Document, Role: p.Role, Since: optionalDate(p.Since)}
	}
	return out
}

func registryMonthOf(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01")
	return &s
}
