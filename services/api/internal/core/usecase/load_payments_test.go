package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

var tceTestHeader = []string{"Ente", "Unidade", "Ano", "Mes", "NumeroEmpenho", "TipoPessoa", "CPFCNPJ", "Funcao", "Empenhado", "Liquidado", "Pago"}

type fakeTCE struct {
	rows   map[int][][]string
	failOn int
}

func (f *fakeTCE) Commitments(_ context.Context, year int, each func(header, row []string) error) (string, error) {
	if year == f.failOn {
		return "", errors.New("status 504")
	}
	for _, r := range f.rows[year] {
		if err := each(tceTestHeader, r); err != nil {
			return "", err
		}
	}
	return "sha", nil
}

type fakePaymentRepo struct{ years map[int][]domain.Payment }

func (r *fakePaymentRepo) Ready(context.Context) error { return nil }
func (r *fakePaymentRepo) ReplaceYear(_ context.Context, _ string, year int, payments []domain.Payment) error {
	if r.years == nil {
		r.years = map[int][]domain.Payment{}
	}
	r.years[year] = payments
	return nil
}

func tceRow(year, kind, doc, paid string) []string {
	return []string{"SAO GONCALO", "PREFEITURA", year, "3", "7", kind, doc, "SAÚDE", "0.0", "0.0", paid}
}

func tceFixture() *fakeTCE {
	return &fakeTCE{rows: map[int][][]string{
		2025: {tceRow("2025", "JURÍDICA", "39818737000151", "100.25"), tceRow("2025", "FÍSICA", "12345678901", "50.0"), tceRow("2025", "JURÍDICA", "39818737000151", "x")},
		2026: {tceRow("2026", "JURÍDICA", "11222333000181", "1.0")},
	}}
}

func paymentsNow() time.Time { return time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC) }

func TestLoadPaymentsDefaultsToThisAndLastYearAndDropsPeople(t *testing.T) {
	repo, raw := &fakePaymentRepo{}, &memObjects{}
	uc := NewLoadPayments(tceFixture(), repo, &memRuns{}, raw, paymentsNow)

	run, err := uc.Execute(context.Background(), 0, 0)

	if err != nil {
		t.Fatal(err)
	}
	if len(repo.years[2025]) != 1 || repo.years[2025][0].PaidCents != 10025 || len(repo.years[2026]) != 1 {
		t.Errorf("anos: %+v", repo.years)
	}
	if run.Found != 3 || run.Stored != 2 || run.Skipped != 1 || run.Failed != 1 || !run.Valid() {
		t.Errorf("coleta: %+v", run)
	}
	body := gunzip(t, raw.data["raw/tce_empenhos/2026/09/26/2025.csv.gz"])
	if !strings.HasPrefix(body, "Ente;") || strings.Contains(body, "12345678901") {
		t.Errorf("bruto: %q", body)
	}
}

func TestLoadPaymentsStopsAtTheFailingYear(t *testing.T) {
	src := tceFixture()
	src.failOn = 2026
	repo, runs := &fakePaymentRepo{}, &memRuns{}

	_, err := NewLoadPayments(src, repo, runs, &memObjects{}, paymentsNow).Execute(context.Background(), 2025, 2026)

	if err == nil || len(repo.years[2025]) != 1 || repo.years[2026] != nil {
		t.Fatalf("err=%v anos=%+v", err, repo.years)
	}
	if len(runs.runs) != 1 || !strings.Contains(runs.runs[0].Error, "504") {
		t.Errorf("coleta: %+v", runs.runs)
	}
}

func TestLoadPaymentsRejectsYearsOutsideTheCoverage(t *testing.T) {
	uc := NewLoadPayments(tceFixture(), &fakePaymentRepo{}, &memRuns{}, &memObjects{}, paymentsNow)

	for _, r := range [][2]int{{2019, 2020}, {2026, 2025}, {2025, 2027}} {
		if _, err := uc.Execute(context.Background(), r[0], r[1]); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("%v: %v", r, err)
		}
	}
}
