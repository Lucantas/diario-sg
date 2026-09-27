package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	SourceSiconfi        = "siconfi"
	LastRREOPeriod       = 6
	FirstRREOYear        = 2017
	LowControlCoverageBP = 9000
	rreoTotalExpenses    = "TotalDespesas"
	rreoCommittedColumn  = "DESPESAS EMPENHADAS ATÉ O BIMESTRE"
	rreoLiquidatedColumn = "DESPESAS LIQUIDADAS ATÉ O BIMESTRE"
	rreoPaidColumn       = "DESPESAS PAGAS ATÉ O BIMESTRE"
	basisPointsWhole     = 10000
)

type FiscalTotal struct {
	Year            int
	Period          int
	CommittedCents  int64
	LiquidatedCents int64
	PaidCents       int64
	SourceURL       string
}

type YearPaid struct {
	Year           int
	CommittedCents int64
	PaidCents      int64
}

type FiscalControl struct {
	FiscalTotal
	TCECommittedCents int64
	TCEPaidCents      int64
	TCELoaded         bool
	PaidCoverageBP    int
}

func (c FiscalControl) LowCoverage() bool {
	return c.TCELoaded && c.Period == LastRREOPeriod && c.PaidCents > 0 && c.PaidCoverageBP < LowControlCoverageBP
}

type rreoResponse struct {
	Items []struct {
		Account string  `json:"cod_conta"`
		Column  string  `json:"coluna"`
		Value   float64 `json:"valor"`
	} `json:"items"`
}

func ParseRREOTotals(body []byte, year, period int) (FiscalTotal, bool, error) {
	var r rreoResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return FiscalTotal{}, false, fmt.Errorf("RREO %d/%d: %w", period, year, err)
	}
	if len(r.Items) == 0 {
		return FiscalTotal{}, false, nil
	}
	t := FiscalTotal{Year: year, Period: period}
	found := 0
	for _, it := range r.Items {
		if it.Account != rreoTotalExpenses {
			continue
		}
		column := strings.ToUpper(it.Column)
		switch {
		case strings.HasPrefix(column, rreoCommittedColumn):
			t.CommittedCents, found = toCents(it.Value), found+1
		case strings.HasPrefix(column, rreoLiquidatedColumn):
			t.LiquidatedCents, found = toCents(it.Value), found+1
		case strings.HasPrefix(column, rreoPaidColumn):
			t.PaidCents, found = toCents(it.Value), found+1
		}
	}
	if found != 3 {
		return FiscalTotal{}, false, fmt.Errorf("RREO %d/%d sem empenhado, liquidado e pago na conta %s", period, year, rreoTotalExpenses)
	}
	return t, true, nil
}

func BuildFiscalControl(totals []FiscalTotal, tce []YearPaid) []FiscalControl {
	byYear := map[int]YearPaid{}
	for _, y := range tce {
		byYear[y.Year] = y
	}
	out := make([]FiscalControl, 0, len(totals))
	for _, t := range totals {
		y, loaded := byYear[t.Year]
		c := FiscalControl{FiscalTotal: t, TCECommittedCents: y.CommittedCents, TCEPaidCents: y.PaidCents, TCELoaded: loaded}
		if loaded && t.PaidCents > 0 {
			c.PaidCoverageBP = int(c.TCEPaidCents * basisPointsWhole / t.PaidCents)
		}
		out = append(out, c)
	}
	return out
}
