package http

import (
	"net/http"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type amendmentPaymentDTO struct {
	Code       string `json:"code"`
	Author     string `json:"author"`
	Kind       string `json:"kind"`
	Month      string `json:"month"`
	CNPJ       string `json:"cnpj"`
	Name       string `json:"name"`
	ValueCents int64  `json:"value_cents"`
}

type amendmentDTO struct {
	Code            string `json:"code"`
	Year            int    `json:"year"`
	Kind            string `json:"kind"`
	Author          string `json:"author"`
	Function        string `json:"function"`
	Action          string `json:"action"`
	CommittedCents  int64  `json:"committed_cents"`
	LiquidatedCents int64  `json:"liquidated_cents"`
	PaidCents       int64  `json:"paid_cents"`
}

type transferTotalDTO struct {
	Year       int    `json:"year"`
	Kind       string `json:"kind"`
	Function   string `json:"function"`
	ValueCents int64  `json:"value_cents"`
}

type favoredDTO struct {
	CNPJ       string   `json:"cnpj"`
	Name       string   `json:"name"`
	Payments   int      `json:"payments"`
	ValueCents int64    `json:"value_cents"`
	Authors    []string `json:"authors"`
	First      string   `json:"first"`
	Last       string   `json:"last"`
}

type specialExecutorDTO struct {
	CNPJ       string `json:"cnpj"`
	Name       string `json:"name"`
	Object     string `json:"object"`
	ValueCents int64  `json:"value_cents"`
}

type specialTransferDTO struct {
	PlanID         int64                `json:"plan_id"`
	Code           string               `json:"code"`
	Year           int                  `json:"year"`
	Status         string               `json:"status"`
	Author         string               `json:"author"`
	Amendment      string               `json:"amendment"`
	Area           string               `json:"area"`
	ValueCents     int64                `json:"value_cents"`
	Executors      []specialExecutorDTO `json:"executors"`
	CommittedCents int64                `json:"committed_cents"`
	PaidCents      int64                `json:"paid_cents"`
	LastPaidAt     *string              `json:"last_paid_at"`
	WorkPlanStatus string               `json:"work_plan_status"`
	ExecutionEnd   *string              `json:"execution_end"`
	ReportKind     string               `json:"report_kind"`
	ReportAt       *string              `json:"report_at"`
	ExecutedCents  int64                `json:"executed_cents"`
	PendingCents   int64                `json:"pending_cents"`
}

type federalDTO struct {
	TransfersCoverage *paymentCoverageDTO  `json:"transfers_coverage"`
	Transfers         []transferTotalDTO   `json:"transfers"`
	Amendments        []amendmentDTO       `json:"amendments"`
	Favored           []favoredDTO         `json:"favored"`
	SpecialTransfers  []specialTransferDTO `json:"special_transfers"`
}

const monthDTOLayout = "2006-01"

func (a *API) federal(w http.ResponseWriter, r *http.Request) {
	rep, err := a.Federal.Execute(r.Context())
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	out := federalDTO{Transfers: make([]transferTotalDTO, len(rep.Transfers)), Amendments: make([]amendmentDTO, len(rep.Amendments)),
		Favored: make([]favoredDTO, len(rep.Favored)), SpecialTransfers: toSpecialTransferDTOs(rep.Special)}
	if rep.TransfersFrom != nil && rep.TransfersTo != nil {
		out.TransfersCoverage = &paymentCoverageDTO{From: rep.TransfersFrom.Format(monthDTOLayout), To: rep.TransfersTo.Format(monthDTOLayout)}
	}
	for i, t := range rep.Transfers {
		out.Transfers[i] = transferTotalDTO{Year: t.Year, Kind: t.Kind, Function: t.Function, ValueCents: t.ValueCents}
	}
	for i, am := range rep.Amendments {
		out.Amendments[i] = amendmentDTO{Code: am.Code, Year: am.Year, Kind: am.Kind, Author: am.Author, Function: am.Function, Action: am.Action,
			CommittedCents: am.CommittedCents, LiquidatedCents: am.LiquidatedCents, PaidCents: am.PaidCents}
	}
	for i, f := range rep.Favored {
		out.Favored[i] = favoredDTO{CNPJ: f.CNPJ, Name: f.Name, Payments: f.Payments, ValueCents: f.ValueCents, Authors: f.Authors,
			First: f.First.Format(monthDTOLayout), Last: f.Last.Format(monthDTOLayout)}
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, out)
}

func toAmendmentPaymentDTOs(payments []domain.AmendmentPayment) []amendmentPaymentDTO {
	out := make([]amendmentPaymentDTO, len(payments))
	for i, p := range payments {
		out[i] = amendmentPaymentDTO{Code: p.Code, Author: p.Author, Kind: p.Kind, Month: p.Month.Format(monthDTOLayout), CNPJ: p.CNPJ,
			Name: p.Name, ValueCents: p.ValueCents}
	}
	return out
}

func toSpecialTransferDTOs(transfers []domain.SpecialTransfer) []specialTransferDTO {
	out := make([]specialTransferDTO, len(transfers))
	for i, st := range transfers {
		dto := specialTransferDTO{PlanID: st.PlanID, Code: st.Code, Year: st.Year, Status: st.Status, Author: st.Author, Amendment: st.Amendment,
			Area: st.Area, ValueCents: st.ValueCents, Executors: make([]specialExecutorDTO, len(st.Executors)), CommittedCents: st.CommittedCents,
			PaidCents: st.PaidCents, LastPaidAt: dayOf(st.LastPaidAt), WorkPlanStatus: st.WorkPlanStatus, ExecutionEnd: dayOf(st.ExecutionEnd),
			ReportKind: st.ReportKind, ReportAt: dayOf(st.ReportAt), ExecutedCents: st.ExecutedCents, PendingCents: st.PendingCents}
		for j, e := range st.Executors {
			dto.Executors[j] = specialExecutorDTO{CNPJ: e.CNPJ, Name: e.Name, Object: e.Object, ValueCents: e.ValueCents}
		}
		out[i] = dto
	}
	return out
}
