package http

import (
	"net/http"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type agentMonthDTO struct {
	Month         string `json:"month"`
	Office        string `json:"office"`
	GrossCents    int64  `json:"gross_cents"`
	DiscountCents *int64 `json:"discount_cents"`
	NetCents      *int64 `json:"net_cents"`
}

type politicalAgentDTO struct {
	Body              string          `json:"body"`
	Role              string          `json:"role"`
	Name              string          `json:"name"`
	Offices           []string        `json:"offices"`
	Party             string          `json:"party,omitempty"`
	ParliamentaryName string          `json:"parliamentary_name,omitempty"`
	First             string          `json:"first"`
	Last              string          `json:"last"`
	Months            []agentMonthDTO `json:"months"`
}

type subsidyNormDTO struct {
	Role       string `json:"role"`
	FromYear   int    `json:"from_year"`
	ToYear     int    `json:"to_year"`
	ValueCents int64  `json:"value_cents"`
	Norm       string `json:"norm"`
	Diario     string `json:"diario"`
	Search     string `json:"search"`
}

type payCoverageDTO struct {
	Body string `json:"body"`
	From string `json:"from"`
	To   string `json:"to"`
}

type politicalAgentsDTO struct {
	Agents   []politicalAgentDTO `json:"agents"`
	Norms    []subsidyNormDTO    `json:"norms"`
	Coverage []payCoverageDTO    `json:"coverage"`
}

func (a *API) politicalAgents(w http.ResponseWriter, r *http.Request) {
	rep, err := a.Agents.Execute(r.Context(), r.URL.Query().Get("role"), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	writeJSON(w, http.StatusOK, toPoliticalAgentsDTO(rep))
}

func toPoliticalAgentsDTO(rep domain.PoliticalAgentsReport) politicalAgentsDTO {
	out := politicalAgentsDTO{Agents: make([]politicalAgentDTO, 0, len(rep.Agents)), Norms: make([]subsidyNormDTO, 0, len(rep.Norms)),
		Coverage: make([]payCoverageDTO, 0, len(rep.Coverage))}
	for _, ag := range rep.Agents {
		dto := politicalAgentDTO{Body: ag.Body, Role: ag.Role, Name: ag.Name, Offices: ag.Offices, Party: ag.Party, ParliamentaryName: ag.ParliamentaryName,
			First: ag.FirstMonth().Format(monthDTOLayout), Last: ag.LastMonth().Format(monthDTOLayout), Months: make([]agentMonthDTO, 0, len(ag.Months))}
		for _, m := range ag.Months {
			dto.Months = append(dto.Months, agentMonthDTO{Month: m.Month.Format(monthDTOLayout), Office: m.Office, GrossCents: m.GrossCents,
				DiscountCents: m.DiscountCents, NetCents: m.NetCents})
		}
		out.Agents = append(out.Agents, dto)
	}
	for _, n := range rep.Norms {
		out.Norms = append(out.Norms, subsidyNormDTO{Role: n.Role, FromYear: n.FromYear, ToYear: n.ToYear, ValueCents: n.Cents, Norm: n.Norm, Diario: n.Diario, Search: n.Search})
	}
	for _, c := range rep.Coverage {
		out.Coverage = append(out.Coverage, payCoverageDTO{Body: c.Body, From: c.From.Format(monthDTOLayout), To: c.To.Format(monthDTOLayout)})
	}
	return out
}
