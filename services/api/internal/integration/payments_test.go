//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

var tceHeader = []string{"Ente", "Unidade", "Ano", "Mes", "NumeroEmpenho", "TipoPessoa", "CPFCNPJ", "Funcao", "Empenhado", "Liquidado", "Pago"}

type tceAPI map[int][][]string

func (a tceAPI) Commitments(_ context.Context, year int, each func(header, row []string) error) (string, error) {
	for _, r := range a[year] {
		if err := each(tceHeader, r); err != nil {
			return "", err
		}
	}
	return "sha", nil
}

func loadPayments(t *testing.T, repo *postgres.PaymentRepo, runs *postgres.FetchRunRepo, api tceAPI, from, to int) {
	t.Helper()
	now := func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }
	if _, err := usecase.NewLoadPayments(api, repo, runs, discardObjects{}, now).Execute(context.Background(), from, to); err != nil {
		t.Fatal(err)
	}
}

func commitment(year, month, unit, number, cnpj, paid string) []string {
	return []string{"SAO GONCALO", unit, year, month, number, "JURÍDICA", cnpj, "EDUCAÇÃO", "0.0", paid, paid}
}

func TestPaymentsReplaceTheYearAndShowInTheCompanyPage(t *testing.T) {
	srv, db := newServerFor(t, semedHomologacao2025)
	repo, runs := postgres.NewPaymentRepo(db), postgres.NewFetchRunRepo(db)

	loadPayments(t, repo, runs, tceAPI{
		2025: {commitment("2025", "03", "FUNDO MUN EDUCACAO SAO GONCALO", "1", fpVieira, "1000.50"), commitment("2025", "01", "PREFEITURA SÃO GONÇALO", "9", fpVieira, "0"), commitment("2025", "04", "PREFEITURA SÃO GONÇALO", "2", fpVieira, "10")},
		2026: {commitment("2026", "01", "PREFEITURA SÃO GONÇALO", "3", fpVieira, "5")},
	}, 2025, 2026)
	loadPayments(t, repo, runs, tceAPI{2026: {commitment("2026", "02", "PREFEITURA SÃO GONÇALO", "4", fpVieira, "7")}}, 2026, 2026)

	var company struct {
		Payments []struct {
			Year      int      `json:"year"`
			Units     []string `json:"units"`
			PaidCents int64    `json:"paid_cents"`
		} `json:"payments"`
		Coverage *struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"payments_coverage"`
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/"+fpVieira, &company)
	if len(company.Payments) != 2 || company.Payments[0].Year != 2026 || company.Payments[0].PaidCents != 700 ||
		company.Payments[1].PaidCents != 101050 || len(company.Payments[1].Units) != 2 {
		t.Fatalf("pagamentos: %+v", company.Payments)
	}
	if company.Coverage == nil || company.Coverage.From != "2025-03" || company.Coverage.To != "2026-02" {
		t.Errorf("cobertura: %+v", company.Coverage)
	}

	var panel struct {
		Items []struct {
			CNPJ      string `json:"cnpj"`
			PaidCents int64  `json:"paid_cents"`
		} `json:"items"`
		Years []struct {
			Year      int   `json:"year"`
			PaidCents int64 `json:"paid_cents"`
		} `json:"years"`
	}
	getJSON(t, srv.URL+"/v1/panels/suppliers", &panel)
	if len(panel.Items) == 0 || panel.Items[0].PaidCents != 101750 {
		t.Errorf("pago no painel: %+v", panel.Items)
	}
	paidIn := map[int]int64{}
	for _, y := range panel.Years {
		paidIn[y.Year] = y.PaidCents
	}
	if paidIn[2025] != 101050 || paidIn[2026] != 700 {
		t.Errorf("pago por ano: %+v", panel.Years)
	}
}

func TestPaymentPatternsAndTheRegistryIncludeUncitedCreditors(t *testing.T) {
	srv, db := newServerFor(t, semedHomologacao2025)
	repo, runs := postgres.NewPaymentRepo(db), postgres.NewFetchRunRepo(db)
	const uncited = "11222333000181"
	loadPayments(t, repo, runs, tceAPI{2025: {commitment("2025", "05", "PREFEITURA SÃO GONÇALO", "1", uncited, "250000")}}, 2025, 2025)

	cited, err := postgres.NewRegistryRepo(db).CitedCNPJs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range cited {
		found = found || c == uncited
	}
	if !found {
		t.Errorf("credor não citado deveria entrar na carga da Receita: %v", cited)
	}

	var res patternsResponse
	getJSON(t, srv.URL+"/v1/patterns", &res)
	for _, item := range res.Items {
		if item.ID != "pago_sem_publicacao" {
			continue
		}
		if len(item.Findings) != 1 || !strings.HasPrefix(item.Findings[0].Title, "CNPJ 11.222.333/0001-81: R$ 250.000,00 pagos de 2025") {
			t.Fatalf("pago sem publicação: %+v", item.Findings)
		}
		return
	}
	t.Fatal("padrão pago_sem_publicacao ausente")
}
