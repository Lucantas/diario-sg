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
		out.Recentes = append(out.Recentes, commitmentOf(c))
	}
	return out
}

func commitmentOf(c domain.MunicipalCommitment) municipalCommitmentDTO {
	return municipalCommitmentDTO{Entidade: c.Entity, Empenho: c.Number + "/" + c.Date.Format("2006"),
		Data: c.Date.Format(time.DateOnly), Processo: c.Process, TipoProcesso: c.ProcessKind, Modalidade: c.Modality, Objeto: c.Object,
		EmpenhadoCentavos: c.CommittedCents, PagoCentavos: c.PaidCents}
}

const muralRowsPerEntity = 10

type muralRowDTO struct {
	Lista         string `json:"lista"`
	Edital        string `json:"edital"`
	Processo      string `json:"processo"`
	Modalidade    string `json:"modalidade"`
	Objeto        string `json:"objeto"`
	Situacao      string `json:"situacao,omitempty"`
	Abertura      string `json:"abertura,omitempty"`
	Instrumento   string `json:"instrumento,omitempty"`
	Fornecedor    string `json:"fornecedor,omitempty"`
	ValorCentavos int64  `json:"valor_centavos,omitempty"`
	URL           string `json:"url"`
}

type muralOfDTO struct {
	Aviso      string        `json:"aviso"`
	Contratos  []muralRowDTO `json:"contratos"`
	Licitacoes []muralRowDTO `json:"licitacoes"`
}

const muralCaveat = "Ligado pelo número do processo dos empenhos da empresa no portal da Prefeitura; o mural não traz CNPJ e o número não tem o órgão, " +
	"então confira o fornecedor e o objeto."

func muralOf(m domain.MuralMatches) *muralOfDTO {
	if len(m.Procurements) == 0 && len(m.Contracts) == 0 {
		return nil
	}
	out := &muralOfDTO{Aviso: muralCaveat, Contratos: []muralRowDTO{}, Licitacoes: []muralRowDTO{}}
	for _, c := range m.Contracts[:min(len(m.Contracts), muralRowsPerEntity)] {
		out.Contratos = append(out.Contratos, muralContractOf(c))
	}
	for _, p := range m.Procurements[:min(len(m.Procurements), muralRowsPerEntity)] {
		out.Licitacoes = append(out.Licitacoes, muralTenderOf(p))
	}
	return out
}

func muralContractOf(c domain.ProcurementContract) muralRowDTO {
	return muralRowDTO{Lista: domain.MuralContracts, Edital: c.Notice, Processo: c.Process, Modalidade: c.Modality,
		Objeto: c.Object, Instrumento: c.Instrument, Fornecedor: c.Supplier, ValorCentavos: c.ValueCents, URL: c.DocumentURL}
}

func muralTenderOf(p domain.Procurement) muralRowDTO {
	row := muralRowDTO{Lista: p.List, Edital: p.Notice, Processo: p.Process, Modalidade: p.Modality, Objeto: p.Object, Situacao: p.Status, URL: p.URL}
	if p.OpensAt != nil {
		row.Abertura = p.OpensAt.Format(time.DateOnly)
	}
	return row
}

type diarioSanctionOutDTO struct {
	Tipo      string `json:"tipo"`
	GazetteID string `json:"edicao_id"`
	Position  int    `json:"posicao"`
	Diario    string `json:"diario"`
	Date      string `json:"data"`
	Pages     string `json:"paginas"`
	Title     string `json:"titulo"`
	URL       string `json:"url"`
	Archived  string `json:"copia_arquivada"`
}

func (s *server) diarioSanctionsOf(sanctions []domain.DiarioSanction) []diarioSanctionOutDTO {
	out := make([]diarioSanctionOutDTO, len(sanctions))
	for i, sn := range sanctions {
		h := sn.Act
		src := sourceOf(citableHit(h), s.webURL)
		out[i] = diarioSanctionOutDTO{Tipo: string(sn.Kind), GazetteID: h.GazetteID, Position: h.Position, Diario: domain.SourceOrDefault(h.Source),
			Date: h.PublishedAt.Format(time.DateOnly), Pages: pageRange(h.PageStart, h.PageEnd), Title: h.Title, URL: src.URL, Archived: src.ArchivedCopy}
	}
	return out
}
