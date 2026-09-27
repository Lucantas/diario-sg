package http

import (
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type municipalYearDTO struct {
	Year           int   `json:"year"`
	Commitments    int   `json:"commitments"`
	CommittedCents int64 `json:"committed_cents"`
	PaidCents      int64 `json:"paid_cents"`
}

type municipalCommitmentDTO struct {
	Entity          string `json:"entity"`
	Year            int    `json:"year"`
	Number          string `json:"number"`
	Date            string `json:"date"`
	Object          string `json:"object"`
	ProcessKind     string `json:"process_kind"`
	Process         string `json:"process"`
	Modality        string `json:"modality"`
	CommittedCents  int64  `json:"committed_cents"`
	LiquidatedCents int64  `json:"liquidated_cents"`
	PaidCents       int64  `json:"paid_cents"`
}

type municipalSupplierDTO struct {
	Years  []municipalYearDTO       `json:"years"`
	Recent []municipalCommitmentDTO `json:"recent"`
}

func toMunicipalSupplierDTO(m domain.MunicipalSupplier) municipalSupplierDTO {
	out := municipalSupplierDTO{Years: make([]municipalYearDTO, len(m.Years)), Recent: make([]municipalCommitmentDTO, len(m.Recent))}
	for i, y := range m.Years {
		out.Years[i] = municipalYearDTO{Year: y.Year, Commitments: y.Commitments, CommittedCents: y.CommittedCents, PaidCents: y.PaidCents}
	}
	for i, c := range m.Recent {
		out.Recent[i] = municipalCommitmentDTO{Entity: c.Entity, Year: c.Year, Number: c.Number, Date: c.Date.Format(time.DateOnly), Object: c.Object,
			ProcessKind: c.ProcessKind, Process: c.Process, Modality: c.Modality, CommittedCents: c.CommittedCents,
			LiquidatedCents: c.LiquidatedCents, PaidCents: c.PaidCents}
	}
	return out
}
