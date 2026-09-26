package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	SourceTCE         = "tce_empenhos"
	RecordPaymentYear = "pagamentos_ano"

	tceLegalEntity = "JURÍDICA"
	centsPerReal   = 100
	byteOrderMark  = "\xef\xbb\xbf"
)

type Payment struct {
	Source          string
	Year            int
	Month           int
	Unit            string
	Commitment      string
	CNPJ            string
	Function        string
	CommittedCents  int64
	LiquidatedCents int64
	PaidCents       int64
}

type PaymentYear struct {
	Year            int
	Units           []string
	CommittedCents  int64
	LiquidatedCents int64
	PaidCents       int64
}

type PaymentCoverage struct {
	FromYear, FromMonth int
	ToYear, ToMonth     int
}

type TCEColumns map[string]int

const (
	tceColUnit       = "Unidade"
	tceColYear       = "Ano"
	tceColMonth      = "Mes"
	tceColCommitment = "NumeroEmpenho"
	tceColKind       = "TipoPessoa"
	tceColDocument   = "CPFCNPJ"
	tceColFunction   = "Funcao"
	tceColCommitted  = "Empenhado"
	tceColLiquidated = "Liquidado"
	tceColPaid       = "Pago"
)

var requiredTCEColumns = []string{tceColUnit, tceColYear, tceColMonth, tceColCommitment, tceColKind, tceColDocument,
	tceColFunction, tceColCommitted, tceColLiquidated, tceColPaid}

func NewTCEColumns(header []string) (TCEColumns, error) {
	cols := TCEColumns{}
	for i, name := range header {
		cols[strings.TrimSpace(strings.TrimPrefix(name, byteOrderMark))] = i
	}
	for _, name := range requiredTCEColumns {
		if _, ok := cols[name]; !ok {
			return nil, fmt.Errorf("cabeçalho do TCE sem a coluna %q", name)
		}
	}
	return cols, nil
}

func (c TCEColumns) get(row []string, name string) string {
	i := c[name]
	if i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func (c TCEColumns) IsCompany(row []string) bool {
	return c.get(row, tceColKind) == tceLegalEntity && len(c.get(row, tceColDocument)) == cnpjDigits
}

func ParseTCERow(c TCEColumns, row []string) (Payment, error) {
	if !c.IsCompany(row) {
		return Payment{}, fmt.Errorf("empenho %s não é de pessoa jurídica", c.get(row, tceColCommitment))
	}
	year, err := strconv.Atoi(c.get(row, tceColYear))
	if err != nil {
		return Payment{}, fmt.Errorf("ano do empenho: %w", err)
	}
	month, err := strconv.Atoi(c.get(row, tceColMonth))
	if err != nil || month < 1 || month > 12 {
		return Payment{}, fmt.Errorf("mês do empenho: %q", c.get(row, tceColMonth))
	}
	p := Payment{Source: SourceTCE, Year: year, Month: month, Unit: squeezed(c.get(row, tceColUnit)),
		Commitment: c.get(row, tceColCommitment), CNPJ: c.get(row, tceColDocument), Function: squeezed(c.get(row, tceColFunction))}
	for _, v := range []struct {
		col string
		dst *int64
	}{{tceColCommitted, &p.CommittedCents}, {tceColLiquidated, &p.LiquidatedCents}, {tceColPaid, &p.PaidCents}} {
		cents, err := parseDecimalCents(c.get(row, v.col))
		if err != nil {
			return Payment{}, fmt.Errorf("%s do empenho %s: %w", v.col, p.Commitment, err)
		}
		*v.dst = cents
	}
	return p, nil
}

func parseDecimalCents(s string) (int64, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(math.Round(f * centsPerReal)), nil
}

type PaidTotal struct {
	CNPJ      string
	Year      int
	Chamber   bool
	PaidCents int64
}

func (t PaidTotal) paidBy(source string) bool {
	return t.Chamber == (source == SourceDiarioCamara)
}
