package http

import (
	"net/http"
	"strings"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type staffGroupDTO struct {
	Group string `json:"group"`
	Label string `json:"label"`
}

type staffTotalDTO struct {
	Headcount         int   `json:"headcount"`
	RemunerationCents int64 `json:"remuneration_cents"`
}

type staffMonthDTO struct {
	Month string `json:"month"`
	staffTotalDTO
	Groups       []staffTotalDTO `json:"groups"`
	Appointments int             `json:"appointments"`
	Dismissals   int             `json:"dismissals"`
}

type staffPanelDTO struct {
	Units        []string        `json:"units"`
	Unit         string          `json:"unit"`
	DiarioSource string          `json:"diario_source"`
	Groups       []staffGroupDTO `json:"groups"`
	Months       []staffMonthDTO `json:"months"`
}

func (a *API) staffPanel(w http.ResponseWriter, r *http.Request) {
	panel, err := a.Staff.Execute(r.Context(), strings.TrimSpace(r.URL.Query().Get("unit")))
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, toStaffPanelDTO(panel))
}

func toStaffPanelDTO(p domain.StaffPanel) staffPanelDTO {
	out := staffPanelDTO{Units: p.Units, Unit: p.Unit, DiarioSource: p.DiarioSource,
		Groups: make([]staffGroupDTO, len(p.Groups)), Months: make([]staffMonthDTO, len(p.Months))}
	if out.Units == nil {
		out.Units = []string{}
	}
	for i, g := range p.Groups {
		out.Groups[i] = staffGroupDTO{Group: g.Group, Label: g.Label}
	}
	for i, m := range p.Months {
		groups := make([]staffTotalDTO, len(m.Groups))
		for j, g := range m.Groups {
			groups[j] = staffTotalDTO{Headcount: g.Headcount, RemunerationCents: g.RemunerationCents}
		}
		out.Months[i] = staffMonthDTO{Month: m.Month.Format("2006-01"), Groups: groups, Appointments: m.Appointments, Dismissals: m.Dismissals,
			staffTotalDTO: staffTotalDTO{Headcount: m.Headcount, RemunerationCents: m.RemunerationCents}}
	}
	return out
}
