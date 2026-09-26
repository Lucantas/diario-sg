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
		ValueCents: 50000000, SignedAt: &signed}
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
