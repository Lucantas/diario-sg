package http

import (
	"net/http"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type stalledWorkDTO struct {
	Contract       string  `json:"contract"`
	CNPJ           string  `json:"cnpj"`
	Contractor     string  `json:"contractor"`
	Organ          string  `json:"organ"`
	Function       string  `json:"function"`
	TotalCents     int64   `json:"total_cents"`
	PaidCents      int64   `json:"paid_cents"`
	StalledAt      *string `json:"stalled_at"`
	StartedAt      *string `json:"started_at"`
	StalledFor     string  `json:"stalled_for"`
	Reason         string  `json:"reason"`
	ContractStatus string  `json:"contract_status"`
	Funding        string  `json:"funding"`
}

type accountDTO struct {
	Year        int    `json:"year"`
	Opinion     string `json:"opinion"`
	Process     string `json:"process"`
	Responsible string `json:"responsible"`
}

type condemnationDTO struct {
	Condemnation string  `json:"condemnation"`
	Year         int     `json:"year"`
	ValueCents   int64   `json:"value_cents"`
	Organ        string  `json:"organ"`
	SessionDate  *string `json:"session_date"`
}

type penaltyProcessDTO struct {
	Process       string            `json:"process"`
	Search        string            `json:"search"`
	Organs        []string          `json:"organs"`
	Natures       []string          `json:"natures"`
	TotalCents    int64             `json:"total_cents"`
	LastSession   *string           `json:"last_session"`
	Condemnations []condemnationDTO `json:"condemnations"`
}

type fiscalControlDTO struct {
	Year              int    `json:"year"`
	Period            int    `json:"period"`
	CommittedCents    int64  `json:"committed_cents"`
	LiquidatedCents   int64  `json:"liquidated_cents"`
	PaidCents         int64  `json:"paid_cents"`
	SourceURL         string `json:"source_url"`
	TCECommittedCents int64  `json:"tce_committed_cents"`
	TCEPaidCents      int64  `json:"tce_paid_cents"`
	TCELoaded         bool   `json:"tce_loaded"`
	PaidCoverageBP    int    `json:"paid_coverage_bp"`
	LowCoverage       bool   `json:"low_coverage"`
	PortalPaidCents   *int64 `json:"portal_paid_cents"`
}

type oversightDTO struct {
	Accounts      []accountDTO        `json:"accounts"`
	Penalties     []penaltyProcessDTO `json:"penalties"`
	Works         []stalledWorkDTO    `json:"works"`
	FiscalControl []fiscalControlDTO  `json:"fiscal_control"`
}

func (a *API) oversight(w http.ResponseWriter, r *http.Request) {
	o, err := a.Oversight.Execute(r.Context())
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	processes := o.Processes
	out := oversightDTO{Accounts: make([]accountDTO, len(o.Accounts)), Penalties: make([]penaltyProcessDTO, len(processes)),
		Works: toStalledWorkDTOs(o.Works), FiscalControl: make([]fiscalControlDTO, len(o.Fiscal))}
	for i, f := range o.Fiscal {
		out.FiscalControl[i] = fiscalControlDTO{Year: f.Year, Period: f.Period, CommittedCents: f.CommittedCents, LiquidatedCents: f.LiquidatedCents,
			PaidCents: f.PaidCents, SourceURL: f.SourceURL, TCECommittedCents: f.TCECommittedCents, TCEPaidCents: f.TCEPaidCents, TCELoaded: f.TCELoaded,
			PaidCoverageBP: f.PaidCoverageBP, LowCoverage: f.LowCoverage()}
		if f.PortalLoaded {
			paid := f.PortalPaidCents
			out.FiscalControl[i].PortalPaidCents = &paid
		}
	}
	for i, acc := range o.Accounts {
		out.Accounts[i] = accountDTO{Year: acc.Year, Opinion: acc.Opinion, Process: acc.Process, Responsible: acc.Responsible}
	}
	for i, p := range processes {
		dto := penaltyProcessDTO{Process: p.Process, Search: p.Search, Organs: p.Organs, Natures: p.Natures, TotalCents: p.TotalCents,
			LastSession: dayOf(p.LastSession), Condemnations: make([]condemnationDTO, len(p.Condemnations))}
		for j, c := range p.Condemnations {
			dto.Condemnations[j] = condemnationDTO{Condemnation: c.Condemnation, Year: c.Year, ValueCents: c.ValueCents, Organ: c.Organ,
				SessionDate: dayOf(c.SessionDate)}
		}
		out.Penalties[i] = dto
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, out)
}

func toStalledWorkDTOs(works []domain.StalledWork) []stalledWorkDTO {
	out := make([]stalledWorkDTO, len(works))
	for i, w := range works {
		out[i] = stalledWorkDTO{Contract: w.Contract, CNPJ: w.CNPJ, Contractor: w.Contractor, Organ: w.Organ, Function: w.Function,
			TotalCents: w.TotalCents, PaidCents: w.PaidCents, StalledAt: dayOf(w.StalledAt), StartedAt: dayOf(w.StartedAt),
			StalledFor: w.StalledFor, Reason: w.Reason, ContractStatus: w.ContractStatus, Funding: w.Funding}
	}
	return out
}
