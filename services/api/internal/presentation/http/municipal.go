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

type procurementDTO struct {
	List      string  `json:"list"`
	ID        int     `json:"id"`
	Notice    string  `json:"notice"`
	Process   string  `json:"process"`
	Modality  string  `json:"modality"`
	Criterion string  `json:"criterion"`
	OpensAt   *string `json:"opens_at"`
	Object    string  `json:"object"`
	Status    string  `json:"status"`
	URL       string  `json:"url"`
}

type procurementContractDTO struct {
	ProcurementID int    `json:"procurement_id"`
	Notice        string `json:"notice"`
	Process       string `json:"process"`
	Modality      string `json:"modality"`
	Object        string `json:"object"`
	ValueCents    int64  `json:"value_cents"`
	Supplier      string `json:"supplier"`
	Instrument    string `json:"instrument"`
	DocumentURL   string `json:"document_url"`
}

type muralDTO struct {
	Procurements []procurementDTO         `json:"procurements"`
	Contracts    []procurementContractDTO `json:"contracts"`
}

func toMuralDTO(m domain.MuralMatches) muralDTO {
	out := muralDTO{Procurements: make([]procurementDTO, len(m.Procurements)), Contracts: make([]procurementContractDTO, len(m.Contracts))}
	for i, p := range m.Procurements {
		out.Procurements[i] = procurementDTO{List: p.List, ID: p.ID, Notice: p.Notice, Process: p.Process, Modality: p.Modality, Criterion: p.Criterion,
			Object: p.Object, Status: p.Status, URL: p.URL}
		if p.OpensAt != nil {
			day := p.OpensAt.Format(time.DateOnly)
			out.Procurements[i].OpensAt = &day
		}
	}
	for i, c := range m.Contracts {
		out.Contracts[i] = procurementContractDTO{ProcurementID: c.ProcurementID, Notice: c.Notice, Process: c.Process, Modality: c.Modality,
			Object: c.Object, ValueCents: c.ValueCents, Supplier: c.Supplier, Instrument: c.Instrument, DocumentURL: c.DocumentURL}
	}
	return out
}
