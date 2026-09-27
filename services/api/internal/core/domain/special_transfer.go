package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	SourceSpecialTransfers = "transferegov_especiais"
	MunicipalityCNPJ       = "28636579000100"
)

type SpecialExecutor struct {
	CNPJ       string
	Name       string
	Object     string
	ValueCents int64
}

type SpecialTransfer struct {
	PlanID         int64
	Code           string
	Year           int
	Status         string
	Author         string
	Amendment      string
	Area           string
	ValueCents     int64
	Executors      []SpecialExecutor
	CommittedCents int64
	PaidCents      int64
	LastPaidAt     *time.Time
	WorkPlanStatus string
	ExecutionEnd   *time.Time
	ReportKind     string
	ReportAt       *time.Time
	ExecutedCents  int64
	PendingCents   int64
}

type SpecialPlanRow struct {
	PlanID     int64   `json:"id_plano_acao"`
	Code       string  `json:"codigo_plano_acao"`
	Year       int     `json:"ano_plano_acao"`
	Status     string  `json:"situacao_plano_acao"`
	Author     string  `json:"nome_parlamentar_emenda_plano_acao"`
	Amendment  string  `json:"numero_emenda_parlamentar_plano_acao"`
	Area       *string `json:"codigo_descricao_areas_politicas_publicas_plano_acao"`
	Custom     float64 `json:"valor_custeio_plano_acao"`
	Investment float64 `json:"valor_investimento_plano_acao"`
}

type SpecialExecutorRow struct {
	PlanID     int64   `json:"id_plano_acao"`
	CNPJ       string  `json:"cnpj_executor"`
	Name       string  `json:"nome_executor"`
	Object     string  `json:"objeto_executor"`
	Custom     float64 `json:"vl_custeio_executor"`
	Investment float64 `json:"vl_investimento_executor"`
}

type SpecialCommitmentRow struct {
	ID     int64   `json:"id_empenho"`
	PlanID int64   `json:"id_plano_acao"`
	Value  float64 `json:"valor_empenho"`
}

type SpecialDocumentRow struct {
	ID           int64   `json:"id_dh"`
	CommitmentID int64   `json:"id_empenho"`
	Value        float64 `json:"valor_dh"`
}

type SpecialOrderRow struct {
	DocumentID int64   `json:"id_dh"`
	BankOrder  *string `json:"numero_ordem_bancaria"`
	IssuedAt   *string `json:"data_emissao_ob"`
}

type SpecialWorkPlanRow struct {
	PlanID int64   `json:"id_plano_acao"`
	Status string  `json:"situacao_plano_trabalho"`
	End    *string `json:"data_fim_execucao_plano_trabalho"`
}

type SpecialReportRow struct {
	PlanID   int64   `json:"id_plano_acao"`
	Kind     string  `json:"tipo_relatorio_gestao_novo"`
	At       string  `json:"data_e_hora_relatorio_gestao_novo"`
	Executed float64 `json:"valor_executado_relatorio_gestao_novo"`
	Pending  float64 `json:"valor_pendente_relatorio_gestao_novo"`
}

type SpecialTransferRows struct {
	Plans       []SpecialPlanRow
	Executors   []SpecialExecutorRow
	Commitments []SpecialCommitmentRow
	Documents   []SpecialDocumentRow
	Orders      []SpecialOrderRow
	WorkPlans   []SpecialWorkPlanRow
	Reports     []SpecialReportRow
}

func decodeRows[T any](body []byte) ([]T, error) {
	var rows []T
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("resposta do Transferegov: %w", err)
	}
	return rows, nil
}

func ParseSpecialPlans(body []byte) ([]SpecialPlanRow, error) {
	return decodeRows[SpecialPlanRow](body)
}

func ParseSpecialExecutors(body []byte) ([]SpecialExecutorRow, error) {
	return decodeRows[SpecialExecutorRow](body)
}

func ParseSpecialCommitments(body []byte) ([]SpecialCommitmentRow, error) {
	return decodeRows[SpecialCommitmentRow](body)
}

func ParseSpecialDocuments(body []byte) ([]SpecialDocumentRow, error) {
	return decodeRows[SpecialDocumentRow](body)
}

func ParseSpecialOrders(body []byte) ([]SpecialOrderRow, error) {
	return decodeRows[SpecialOrderRow](body)
}

func ParseSpecialWorkPlans(body []byte) ([]SpecialWorkPlanRow, error) {
	return decodeRows[SpecialWorkPlanRow](body)
}

func ParseSpecialReports(body []byte) ([]SpecialReportRow, error) {
	return decodeRows[SpecialReportRow](body)
}

func PlanIDs(rows []SpecialPlanRow) []int64 {
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.PlanID
	}
	return ids
}

func CommitmentIDs(rows []SpecialCommitmentRow) []int64 {
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids
}

func DocumentIDs(rows []SpecialDocumentRow) []int64 {
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids
}

func SpecialFilterIn(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return "in.(" + strings.Join(parts, ",") + ")"
}

func BuildSpecialTransfers(rows SpecialTransferRows) []SpecialTransfer {
	out := make([]SpecialTransfer, len(rows.Plans))
	index := make(map[int64]*SpecialTransfer, len(rows.Plans))
	for i, p := range rows.Plans {
		out[i] = SpecialTransfer{PlanID: p.PlanID, Code: p.Code, Year: p.Year, Status: p.Status, Author: strings.TrimSpace(p.Author),
			Amendment: p.Amendment, ValueCents: reaisToCents(p.Custom) + reaisToCents(p.Investment)}
		if p.Area != nil {
			out[i].Area = strings.TrimSpace(*p.Area)
		}
		index[p.PlanID] = &out[i]
	}
	for _, e := range rows.Executors {
		if st := index[e.PlanID]; st != nil {
			st.Executors = append(st.Executors, SpecialExecutor{CNPJ: e.CNPJ, Name: strings.TrimSpace(e.Name), Object: strings.TrimSpace(e.Object),
				ValueCents: reaisToCents(e.Custom) + reaisToCents(e.Investment)})
		}
	}
	planOfCommitment := map[int64]int64{}
	for _, c := range rows.Commitments {
		planOfCommitment[c.ID] = c.PlanID
		if st := index[c.PlanID]; st != nil {
			st.CommittedCents += reaisToCents(c.Value)
		}
	}
	addPayments(rows, index, planOfCommitment)
	for _, w := range rows.WorkPlans {
		if st := index[w.PlanID]; st != nil {
			st.WorkPlanStatus, st.ExecutionEnd = w.Status, dayPtr(w.End)
		}
	}
	for _, r := range rows.Reports {
		st := index[r.PlanID]
		at := dayPtr(&r.At)
		if st == nil || at == nil || (st.ReportAt != nil && !at.After(*st.ReportAt)) {
			continue
		}
		st.ReportKind, st.ReportAt, st.ExecutedCents, st.PendingCents = r.Kind, at, reaisToCents(r.Executed), reaisToCents(r.Pending)
	}
	return out
}

func addPayments(rows SpecialTransferRows, index map[int64]*SpecialTransfer, planOfCommitment map[int64]int64) {
	paidOn := map[int64]time.Time{}
	for _, o := range rows.Orders {
		if o.BankOrder == nil || strings.TrimSpace(*o.BankOrder) == "" {
			continue
		}
		if day := dayPtr(o.IssuedAt); day != nil {
			paidOn[o.DocumentID] = *day
		}
	}
	for _, d := range rows.Documents {
		day, paid := paidOn[d.ID]
		st := index[planOfCommitment[d.CommitmentID]]
		if !paid || st == nil {
			continue
		}
		st.PaidCents += reaisToCents(d.Value)
		if st.LastPaidAt == nil || day.After(*st.LastPaidAt) {
			st.LastPaidAt = &day
		}
	}
}

func dayPtr(s *string) *time.Time {
	if s == nil || len(*s) < len(time.DateOnly) {
		return nil
	}
	t, err := time.Parse(time.DateOnly, (*s)[:len(time.DateOnly)])
	if err != nil {
		return nil
	}
	return &t
}
