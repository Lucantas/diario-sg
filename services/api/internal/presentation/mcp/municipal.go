package mcp

import (
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const municipalCommitmentsPerEntity = 10

type municipalYearDTO struct {
	Ano               int   `json:"ano"`
	Empenhos          int   `json:"empenhos"`
	EmpenhadoCentavos int64 `json:"empenhado_centavos"`
	PagoCentavos      int64 `json:"pago_centavos"`
}

type municipalCommitmentDTO struct {
	Entidade          string `json:"entidade"`
	Empenho           string `json:"empenho"`
	Data              string `json:"data"`
	Processo          string `json:"processo"`
	TipoProcesso      string `json:"tipo_processo"`
	Modalidade        string `json:"modalidade"`
	Objeto            string `json:"objeto"`
	EmpenhadoCentavos int64  `json:"empenhado_centavos"`
	PagoCentavos      int64  `json:"pago_centavos"`
}

type municipalSupplierDTO struct {
	PorAno   []municipalYearDTO       `json:"por_ano"`
	Recentes []municipalCommitmentDTO `json:"recentes"`
}

func municipalOf(m domain.MunicipalSupplier) *municipalSupplierDTO {
	if len(m.Years) == 0 {
		return nil
	}
	out := &municipalSupplierDTO{PorAno: make([]municipalYearDTO, len(m.Years))}
	for i, y := range m.Years {
		out.PorAno[i] = municipalYearDTO{Ano: y.Year, Empenhos: y.Commitments, EmpenhadoCentavos: y.CommittedCents, PagoCentavos: y.PaidCents}
	}
	for _, c := range m.Recent[:min(len(m.Recent), municipalCommitmentsPerEntity)] {
		out.Recentes = append(out.Recentes, municipalCommitmentDTO{Entidade: c.Entity, Empenho: c.Number + "/" + c.Date.Format("2006"),
			Data: c.Date.Format(time.DateOnly), Processo: c.Process, TipoProcesso: c.ProcessKind, Modalidade: c.Modality, Objeto: c.Object,
			EmpenhadoCentavos: c.CommittedCents, PagoCentavos: c.PaidCents})
	}
	return out
}
