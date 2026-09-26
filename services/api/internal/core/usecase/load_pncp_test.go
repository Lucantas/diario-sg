package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakePNCP struct{ failOrg string }

func (f *fakePNCP) Contracts(_ context.Context, org string, year int, each func(domain.PNCPContract, bool) error) (string, error) {
	if org == f.failOrg {
		return "", errors.New("status 503")
	}
	if org != "28636579000100" || year != 2025 {
		return "vazio", nil
	}
	if err := each(domain.PNCPContract{ControlNumber: "pj", SupplierCNPJ: "15106169000106"}, true); err != nil {
		return "", err
	}
	return "sha", each(domain.PNCPContract{ControlNumber: "pf", SupplierCNPJ: "12345678901"}, false)
}

type fakePNCPRepo struct {
	from, to  int
	contracts []domain.PNCPContract
	replaced  bool
}

func (r *fakePNCPRepo) Ready(context.Context) error { return nil }
func (r *fakePNCPRepo) ReplaceYears(_ context.Context, from, to int, contracts []domain.PNCPContract) error {
	r.from, r.to, r.contracts, r.replaced = from, to, contracts, true
	return nil
}

func TestLoadPNCPKeepsCompanySuppliersOfEveryMunicipalBody(t *testing.T) {
	repo, raw := &fakePNCPRepo{}, &memObjects{}

	run, err := NewLoadPNCP(&fakePNCP{}, repo, &memRuns{}, raw, paymentsNow).Execute(context.Background(), 0, 0)

	if err != nil {
		t.Fatal(err)
	}
	if repo.from != 2021 || repo.to != 2026 || len(repo.contracts) != 1 || repo.contracts[0].ControlNumber != "pj" {
		t.Fatalf("troca: %d-%d %+v", repo.from, repo.to, repo.contracts)
	}
	if run.Found != 2 || run.Stored != 1 || run.Skipped != 1 || !run.Valid() {
		t.Errorf("coleta: %+v", run)
	}
	if body := gunzip(t, raw.data["raw/pncp_contratos/2026/09/26/contratos.jsonl.gz"]); strings.Contains(body, "12345678901") || !strings.Contains(body, "15106169000106") {
		t.Errorf("bruto: %q", body)
	}
}

func TestLoadPNCPFailureReplacesNothing(t *testing.T) {
	repo, runs := &fakePNCPRepo{}, &memRuns{}

	_, err := NewLoadPNCP(&fakePNCP{failOrg: "29846003000122"}, repo, runs, &memObjects{}, paymentsNow).Execute(context.Background(), 2025, 2025)

	if err == nil || repo.replaced || len(runs.runs) != 1 || !strings.Contains(runs.runs[0].Error, "503") {
		t.Fatalf("falha: %v %v %+v", err, repo.replaced, runs.runs)
	}
}
