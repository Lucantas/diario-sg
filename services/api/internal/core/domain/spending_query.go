package domain

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	lastQueryYear  = 2100
	maxQueryText   = 200
	firstQueryYear = 2000
)

var organSuffixRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]{1,19}$`)

type YearRange struct {
	From int
	To   int
}

func (r YearRange) Empty() bool { return r.From == 0 && r.To == 0 }

func NewYearRange(from, to int) (YearRange, error) {
	for _, y := range []int{from, to} {
		if y != 0 && (y < firstQueryYear || y > lastQueryYear) {
			return YearRange{}, fmt.Errorf("%w: ano %d fora de %d a %d", ErrInvalidInput, y, firstQueryYear, lastQueryYear)
		}
	}
	if from != 0 && to != 0 && from > to {
		return YearRange{}, fmt.Errorf("%w: de (%d) depois de ate (%d)", ErrInvalidInput, from, to)
	}
	return YearRange{From: from, To: to}, nil
}

type PaymentQuery struct {
	CNPJ       string
	Entity     string
	ProcessKey string
	Text       string
	Years      YearRange
	Offset     int
}

func (q PaymentQuery) OnlyYears() bool {
	return q.CNPJ == "" && q.Entity == "" && q.ProcessKey == "" && q.Text == ""
}

func NewPaymentQuery(cnpj, entity, process, text string, from, to, offset int) (PaymentQuery, error) {
	var q PaymentQuery
	var err error
	if q.CNPJ, q.ProcessKey, err = spendingKeys(cnpj, process); err != nil {
		return q, err
	}
	if q.Entity, err = queryText("entidade", entity); err != nil {
		return q, err
	}
	if q.Text, err = queryText("texto", text); err != nil {
		return q, err
	}
	if q.Years, err = NewYearRange(from, to); err != nil {
		return q, err
	}
	if q.Offset, err = queryOffset(offset); err != nil {
		return q, err
	}
	if q.OnlyYears() && q.Years.Empty() {
		return q, fmt.Errorf("%w: informe cnpj, entidade, processo, texto ou os anos (de, ate)", ErrInvalidInput)
	}
	return q, nil
}

type ProcurementQuery struct {
	CNPJ       string
	ProcessKey string
	Text       string
	Organ      string
	Years      YearRange
	Offset     int
}

func NewProcurementQuery(cnpj, process, text, organ string, from, to, offset int) (ProcurementQuery, error) {
	var q ProcurementQuery
	var err error
	if q.CNPJ, q.ProcessKey, err = spendingKeys(cnpj, process); err != nil {
		return q, err
	}
	if q.Text, err = queryText("texto", text); err != nil {
		return q, err
	}
	if q.Organ = strings.ToUpper(strings.TrimSpace(organ)); q.Organ != "" && !organSuffixRe.MatchString(q.Organ) {
		return q, fmt.Errorf("%w: orgao é a sigla do edital, como PMSG, FMS ou FMAS", ErrInvalidInput)
	}
	if q.Years, err = NewYearRange(from, to); err != nil {
		return q, err
	}
	if q.Offset, err = queryOffset(offset); err != nil {
		return q, err
	}
	if q.CNPJ == "" && q.ProcessKey == "" && q.Text == "" && q.Organ == "" && q.Years.Empty() {
		return q, fmt.Errorf("%w: informe cnpj, processo, texto, orgao ou os anos (de, ate)", ErrInvalidInput)
	}
	return q, nil
}

func spendingKeys(cnpj, process string) (string, string, error) {
	var cnpjKey, processKey string
	if strings.TrimSpace(cnpj) != "" {
		n, err := ParseEntityInput(EntityCNPJ, cnpj)
		if err != nil {
			return "", "", err
		}
		cnpjKey = n
	}
	if strings.TrimSpace(process) != "" {
		if processKey = ProcessKey(process); processKey == "" {
			return "", "", fmt.Errorf("%w: processo como número/ano, por exemplo 17943/2024", ErrInvalidInput)
		}
	}
	return cnpjKey, processKey, nil
}

func queryText(field, s string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) > maxQueryText {
		return "", fmt.Errorf("%w: %s com mais de %d caracteres", ErrInvalidInput, field, maxQueryText)
	}
	if s != "" && len([]rune(s)) < 3 {
		return "", fmt.Errorf("%w: %s com menos de 3 caracteres", ErrInvalidInput, field)
	}
	return s, nil
}

func queryOffset(offset int) (int, error) {
	if offset < 0 {
		return 0, fmt.Errorf("%w: pular não pode ser negativo", ErrInvalidInput)
	}
	return offset, nil
}

type SpendingTotals struct {
	Commitments     int
	CommittedCents  int64
	LiquidatedCents int64
	PaidCents       int64
}

type SpendingGroup struct {
	Key  string
	Name string
	SpendingTotals
}

type SpendingYear struct {
	Year int
	SpendingTotals
	ControlPaidCents *int64
}

type PaymentReport struct {
	Totals      SpendingTotals
	ByYear      []SpendingYear
	ByEntity    []SpendingGroup
	BySupplier  []SpendingGroup
	Commitments []MunicipalCommitment
}

func WithFiscalControl(years []SpendingYear, totals []FiscalTotal) []SpendingYear {
	paid := map[int]int64{}
	for _, t := range totals {
		if t.Period == LastRREOPeriod {
			paid[t.Year] = t.PaidCents
		}
	}
	out := make([]SpendingYear, len(years))
	for i, y := range years {
		out[i] = y
		if p, ok := paid[y.Year]; ok {
			out[i].ControlPaidCents = &p
		}
	}
	return out
}

type ProcurementReport struct {
	ProcurementsTotal  int
	Procurements       []Procurement
	ContractsTotal     int
	ContractValueCents int64
	Contracts          []ProcurementContract
	PNCP               []PNCPContract
}
