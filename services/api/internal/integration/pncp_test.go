//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

type pncpAPI map[int][]domain.PNCPContract

func (a pncpAPI) Contracts(_ context.Context, org string, year int, each func(domain.PNCPContract, bool) error) (string, error) {
	if org != domain.MunicipalOrgCNPJs()[0] {
		return "", nil
	}
	for _, c := range a[year] {
		if err := each(c, true); err != nil {
			return "", err
		}
	}
	return "sha", nil
}

func pncpContract(year, seq int, cnpj string) domain.PNCPContract {
	signed := time.Date(year, 3, 10, 0, 0, 0, 0, time.UTC)
	return domain.PNCPContract{ControlNumber: fmt.Sprintf("%s-2-%06d/%d", domain.MunicipalOrgCNPJs()[0], seq, year),
		OrgCNPJ: domain.MunicipalOrgCNPJs()[0], UnitName: "SECRETARIA MUNICIPAL DE EDUCAÇÃO", Year: year, Sequence: seq, Kind: "Contrato",
		Process: "1234/2025", Number: "10/2025", SupplierCNPJ: cnpj, SupplierName: "F.P. VIEIRA ENGENHARIA LTDA", Object: "OBRA",
		ValueCents: 50000000, SignedAt: &signed, PublishedAt: &signed}
}

func loadPNCP(t *testing.T, db *sql.DB, api pncpAPI, from, to int) {
	t.Helper()
	now := func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }
	uc := usecase.NewLoadPNCP(api, postgres.NewPNCPRepo(db), postgres.NewFetchRunRepo(db), discardObjects{}, now)
	if _, err := uc.Execute(context.Background(), from, to); err != nil {
		t.Fatal(err)
	}
}

func TestPNCPReplacesTheYearsReadAndLinksTheSupplier(t *testing.T) {
	_, db := newServerFor(t, semedHomologacao2025)

	loadPNCP(t, db, pncpAPI{2025: {pncpContract(2025, 1, fpVieira)}, 2026: {pncpContract(2026, 1, fpVieira)}}, 2025, 2026)
	loadPNCP(t, db, pncpAPI{2026: {pncpContract(2026, 2, fpVieira)}}, 2026, 2026)

	got, err := postgres.NewPNCPRepo(db).PNCPContractsBySupplier(context.Background(), fpVieira)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Sequence != 2 || got[0].Year != 2026 || got[1].Year != 2025 {
		t.Fatalf("contratos: %+v", got)
	}
	var links int
	if err := db.QueryRow(`SELECT count(*) FROM entity_links WHERE source = $1 AND certainty = 'exata'`, domain.SourcePNCP).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if links != 2 {
		t.Fatalf("ligações: %d", links)
	}
}

func TestPNCPContractsShowInTheCompanyPageAndInThePattern(t *testing.T) {
	srv, db := newServerFor(t, semedHomologacao2025)
	uncited := pncpContract(2026, 3, "99888777000166")
	uncited.Process = "987654/2026"
	loadPNCP(t, db, pncpAPI{2026: {pncpContract(2026, 1, fpVieira), uncited}}, 2026, 2026)

	var company struct {
		PNCP []struct {
			URL        string `json:"url"`
			ValueCents int64  `json:"value_cents"`
			SignedAt   string `json:"signed_at"`
		} `json:"pncp_contracts"`
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/"+fpVieira, &company)
	if len(company.PNCP) != 1 || company.PNCP[0].URL != "https://pncp.gov.br/app/contratos/28636579000100/2026/1" ||
		company.PNCP[0].ValueCents != 50000000 || company.PNCP[0].SignedAt != "2026-03-10" {
		t.Fatalf("contratos do PNCP: %+v", company.PNCP)
	}

	var patterns struct {
		Items []struct {
			ID       string `json:"id"`
			Findings []struct {
				Title string `json:"title"`
				Link  *struct {
					URL string `json:"url"`
				} `json:"link"`
			} `json:"findings"`
		} `json:"items"`
	}
	getJSON(t, srv.URL+"/v1/patterns", &patterns)
	pncp := patterns.Items[0]
	for _, item := range patterns.Items {
		if item.ID == "pncp_sem_extrato" {
			pncp = item
		}
	}
	if pncp.ID != "pncp_sem_extrato" || len(pncp.Findings) != 1 || pncp.Findings[0].Link == nil ||
		pncp.Findings[0].Link.URL != uncited.URL() {
		t.Fatalf("padrão do PNCP: %+v", pncp)
	}
}

func TestPNCPPartialLoadKeepsContractsPublishedInOtherYears(t *testing.T) {
	_, db := newServerFor(t, semedHomologacao2025)
	late := pncpContract(2024, 7, fpVieira)
	published := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	late.PublishedAt = &published

	loadPNCP(t, db, pncpAPI{2024: {pncpContract(2024, 1, fpVieira)}, 2025: {late}}, 2024, 2025)
	loadPNCP(t, db, pncpAPI{2024: {pncpContract(2024, 1, fpVieira)}}, 2024, 2024)

	got, err := postgres.NewPNCPRepo(db).PNCPContractsBySupplier(context.Background(), fpVieira)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("o contrato de 2024 publicado em 2025 sumiu: %+v", got)
	}
}
