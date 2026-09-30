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
	month   string
	name    string
	opened  string
	partner string
}

func (s registryShare) openedAt() string {
	if s.opened == "" {
		return "20110815"
	}
	return s.opened
}

func (s registryShare) partnerName() string {
	if s.partner == "" {
		return "SÓCIA EXEMPLO"
	}
	return s.partner
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
		"Estabelecimentos0.zip": {{"14180324", "0001", "63", "1", "FP VIEIRA", "02", "20110815", "00", "", "", s.openedAt(), "4120400",
			"7112000", "RUA", "EXEMPLO", "10", "", "CENTRO", "24000000", "RJ", "5869", "", "", "", "", "", "", "", "", ""}},
		"Socios0.zip": {{"14180324", "2", s.partnerName(), "***123456**", "49", "20110815", "", "", "", "00", "5"}},
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

func TestRegistryShowsInTheCompanyPagePanelAndMCP(t *testing.T) {
	srv, db := newServerFor(t, semedHomologacao2025)
	loadRegistry(t, postgres.NewRegistryRepo(db), postgres.NewFetchRunRepo(db), registryShare{month: "2026-09", name: "F.P. VIEIRA ENGENHARIA LTDA"})

	var company struct {
		Registry *struct {
			Month    string `json:"month"`
			Name     string `json:"name"`
			Status   string `json:"status"`
			Partners []struct {
				Kind     string `json:"kind"`
				Document string `json:"document"`
			} `json:"partners"`
		} `json:"registry"`
		RegistryMonth *string `json:"registry_month"`
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/14180324000163", &company)
	if company.Registry == nil || company.Registry.Name != "F.P. VIEIRA ENGENHARIA LTDA" || company.Registry.Month != "2026-09" ||
		len(company.Registry.Partners) != 1 || company.Registry.Partners[0].Kind != "pessoa_fisica" || company.Registry.Partners[0].Document != "***123456**" {
		t.Fatalf("cadastro na página da empresa: %+v", company.Registry)
	}

	var unknown struct {
		Registry      *struct{} `json:"registry"`
		RegistryMonth *string   `json:"registry_month"`
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/11222333000181", &unknown)
	if unknown.Registry != nil || unknown.RegistryMonth == nil || *unknown.RegistryMonth != "2026-09" {
		t.Errorf("CNPJ sem cadastro: %+v", unknown)
	}

	var panel struct {
		Items []struct {
			CNPJ string `json:"cnpj"`
			Name string `json:"name"`
		} `json:"items"`
	}
	getJSON(t, srv.URL+"/v1/panels/suppliers", &panel)
	if len(panel.Items) == 0 || panel.Items[0].Name != "F.P. VIEIRA ENGENHARIA LTDA" {
		t.Errorf("nome no painel: %+v", panel.Items)
	}

	code, key := issueKey(t, srv.URL, "")
	if code != 201 {
		t.Fatalf("emissão da chave: %d", code)
	}
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	out, _ := call[struct {
		Registry *struct {
			RazaoSocial string `json:"razao_social"`
			Socios      []struct {
				Documento string `json:"documento"`
			} `json:"socios"`
		} `json:"cadastro_receita"`
	}](t, session, "entidade", map[string]any{"numero": "14.180.324/0001-63"})
	if out.Registry == nil || out.Registry.RazaoSocial != "F.P. VIEIRA ENGENHARIA LTDA" || len(out.Registry.Socios) != 1 {
		t.Errorf("cadastro no MCP: %+v", out.Registry)
	}
}
