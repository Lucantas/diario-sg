package domain

import (
	"errors"
	"testing"
)

func TestNewPaymentQueryNormalizesFilters(t *testing.T) {
	q, err := NewPaymentQuery("12.345.678/0001-95", "  saude  ", "06.10981/2025-7", "", 2024, 2025, 20)

	if err != nil {
		t.Fatal(err)
	}
	if q.CNPJ != "12345678000195" || q.Entity != "saude" || q.ProcessKey != "10981/2025" || q.Years != (YearRange{2024, 2025}) || q.Offset != 20 {
		t.Fatalf("%+v", q)
	}
}

func TestNewPaymentQueryRejectsBadInput(t *testing.T) {
	cases := map[string]func() error{
		"sem filtro":       func() error { _, err := NewPaymentQuery("", "", "", "", 0, 0, 0); return err },
		"anos invertidos":  func() error { _, err := NewPaymentQuery("", "", "", "", 2025, 2024, 0); return err },
		"ano fora":         func() error { _, err := NewPaymentQuery("", "", "", "", 1990, 0, 0); return err },
		"pular negativo":   func() error { _, err := NewPaymentQuery("", "saude", "", "", 0, 0, -1); return err },
		"processo sem ano": func() error { _, err := NewPaymentQuery("", "", "12345", "", 0, 0, 0); return err },
		"texto curto":      func() error { _, err := NewPaymentQuery("", "", "", "ab", 0, 0, 0); return err },
	}
	for name, run := range cases {
		if err := run(); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := NewPaymentQuery("123", "", "", "", 0, 0, 0); !errors.Is(err, ErrInvalidCNPJ) {
		t.Errorf("cnpj inválido: %v", err)
	}
}

func TestPaymentQueryOnlyYearsWhenNoOtherFilter(t *testing.T) {
	years, _ := NewPaymentQuery("", "", "", "", 2024, 0, 0)
	entity, _ := NewPaymentQuery("", "educacao", "", "", 2024, 0, 0)

	if !years.OnlyYears() || entity.OnlyYears() {
		t.Fatalf("%v %v", years.OnlyYears(), entity.OnlyYears())
	}
}

func TestNewProcurementQueryValidatesOrgan(t *testing.T) {
	q, err := NewProcurementQuery("", "", "", " fms ", 0, 0, 0)
	if err != nil || q.Organ != "FMS" {
		t.Fatalf("%+v %v", q, err)
	}

	if _, err := NewProcurementQuery("", "", "", "FMS'; --", 0, 0, 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
	if _, err := NewProcurementQuery("", "", "", "", 0, 0, 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
}

func TestWithFiscalControlUsesOnlyTheYearEndReport(t *testing.T) {
	years := []SpendingYear{{Year: 2025}, {Year: 2026}}
	totals := []FiscalTotal{{Year: 2025, Period: LastRREOPeriod, PaidCents: 900}, {Year: 2026, Period: 4, PaidCents: 300}}

	got := WithFiscalControl(years, totals)

	if got[0].ControlPaidCents == nil || *got[0].ControlPaidCents != 900 || got[1].ControlPaidCents != nil {
		t.Fatalf("%+v", got)
	}
	if years[0].ControlPaidCents != nil {
		t.Fatal("alterou a entrada")
	}
}
