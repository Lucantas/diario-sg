//go:build integration

package integration

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const fpVieira = "14180324000163"

type registryShare struct {
	month string
	name  string
}

func (s registryShare) LatestMonth(context.Context) (string, error) { return s.month, nil }
func (s registryShare) Files(context.Context, string) ([]string, error) {
	return []string{"Empresas0.zip", "Estabelecimentos0.zip", "Socios0.zip"}, nil
}
func (s registryShare) Codes(context.Context, string) (domain.RegistryCodes, error) {
	return domain.RegistryCodes{Natures: map[string]string{"2062": "Sociedade Empresária Limitada"}, Roles: map[string]string{"49": "Sócio-Administrador"}}, nil
}
func (s registryShare) Rows(_ context.Context, _, file string, each func([]string) error) (string, error) {
	rows := map[string][][]string{
		"Empresas0.zip": {{"14180324", s.name, "2062", "49", "500000,00", "03", ""}},
		"Estabelecimentos0.zip": {{"14180324", "0001", "63", "1", "FP VIEIRA", "02", "20110815", "00", "", "", "20110815", "4120400",
			"7112000", "RUA", "EXEMPLO", "10", "", "CENTRO", "24000000", "RJ", "5869", "", "", "", "", "", "", "", "", ""}},
		"Socios0.zip": {{"14180324", "2", "SÓCIA EXEMPLO", "***123456**", "49", "20110815", "", "", "", "00", "5"}},
	}[file]
	for _, r := range rows {
		if err := each(r); err != nil {
			return "", err
		}
	}
	return "sha", nil
}

type discardObjects struct{}

func (discardObjects) Put(_ context.Context, _, _ string, body io.Reader) error {
	_, err := io.Copy(io.Discard, body)
	return err
}

func loadRegistry(t *testing.T, repo *postgres.RegistryRepo, runs *postgres.FetchRunRepo, share registryShare) domain.FetchRun {
	t.Helper()
	run, err := usecase.NewLoadRegistry(share, repo, runs, discardObjects{}, time.Now).Execute(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func TestRegistryLoadReplacesTheMonthAndLinksTheCNPJ(t *testing.T) {
	_, db := newServerFor(t, semedHomologacao2025)
	ctx := context.Background()
	repo, runs := postgres.NewRegistryRepo(db), postgres.NewFetchRunRepo(db)

	loadRegistry(t, repo, runs, registryShare{month: "2026-08", name: "F.P. VIEIRA ENGENHARIA LTDA - ANTIGO"})
	run := loadRegistry(t, repo, runs, registryShare{month: "2026-09", name: "F.P. VIEIRA ENGENHARIA LTDA"})

	if run.Found < 1 || run.Stored != 1 {
		t.Errorf("coleta: %+v", run)
	}
	reg, err := repo.RegistryByCNPJ(ctx, fpVieira)
	if err != nil || reg == nil {
		t.Fatalf("cadastro: %+v %v", reg, err)
	}
	if reg.Company.Name != "F.P. VIEIRA ENGENHARIA LTDA" || reg.Company.CapitalCents != 50000000 || !reg.Month.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("empresa: %+v mês %v", reg.Company, reg.Month)
	}
	if reg.Establishment.Status != "Ativa" || len(reg.Establishment.OtherActivities) != 1 || len(reg.Partners) != 1 || reg.Partners[0].Role != "Sócio-Administrador" {
		t.Errorf("estabelecimento e sócios: %+v %+v", reg.Establishment, reg.Partners)
	}
	var companies, links int
	if err := db.QueryRow(`SELECT count(*) FROM rf_companies`).Scan(&companies); err != nil || companies != 1 {
		t.Errorf("o mês anterior deveria ser trocado: %d %v", companies, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM entity_links WHERE source = 'receita_cnpj' AND certainty = 'exata' AND evidence = 'F.P. VIEIRA ENGENHARIA LTDA'`).Scan(&links); err != nil || links != 1 {
		t.Errorf("ligação do CNPJ: %d %v", links, err)
	}
	names, err := repo.NamesByCNPJ(ctx, []string{fpVieira, "00000000000000"})
	if err != nil || names[fpVieira] != "F.P. VIEIRA ENGENHARIA LTDA" || len(names) != 1 {
		t.Errorf("nomes: %v %v", names, err)
	}
	if month, err := repo.RegistryMonth(ctx); err != nil || month == nil || month.Month() != time.September {
		t.Errorf("mês do cadastro: %v %v", month, err)
	}
	if missing, err := repo.RegistryByCNPJ(ctx, "00000000000000"); err != nil || missing != nil {
		t.Errorf("CNPJ sem cadastro: %+v %v", missing, err)
	}
}
