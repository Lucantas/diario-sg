//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

var portalHeader = []string{"CADASTRO", "CÓDIGO DA SANÇÃO", "TIPO DE PESSOA", "CPF OU CNPJ DO SANCIONADO", "NOME DO SANCIONADO",
	"RAZÃO SOCIAL - CADASTRO RECEITA", "NÚMERO DO PROCESSO", "CATEGORIA DA SANÇÃO", "DATA INÍCIO SANÇÃO", "DATA FINAL SANÇÃO",
	"DATA PUBLICAÇÃO", "ABRAGÊNCIA DA SANÇÃO", "ÓRGÃO SANCIONADOR", "UF ÓRGÃO SANCIONADOR", "ESFERA ÓRGÃO SANCIONADOR", "FUNDAMENTAÇÃO LEGAL"}

type portal struct {
	day   time.Time
	ceis  [][]string
	cepim [][]string
}

func (p portal) LatestDay(context.Context, string) (time.Time, error) { return p.day, nil }
func (p portal) Rows(_ context.Context, register string, _ time.Time, each func(header, row []string) error) (string, error) {
	header, rows := portalHeader, p.ceis
	switch register {
	case domain.RegisterCEPIM:
		header, rows = []string{"CNPJ ENTIDADE", "NOME ENTIDADE", "NÚMERO CONVÊNIO", "ÓRGÃO CONCEDENTE", "MOTIVO DO IMPEDIMENTO"}, p.cepim
	case domain.RegisterCNEP:
		rows = nil
	}
	for _, r := range rows {
		if err := each(header, r); err != nil {
			return "", err
		}
	}
	return "sha", nil
}

func ceisSanction(code, cnpj string) []string {
	return []string{"CEIS", code, "J", cnpj, "F P VIEIRA", "F.P. VIEIRA ENGENHARIA LTDA", "1/2025", "Suspensão", "01/01/2025",
		"01/01/2027", "02/01/2025", "No órgão sancionador", "PREFEITURA X", "RJ", "MUNICIPAL", "LEI 14133"}
}

func loadSanctions(t *testing.T, repo *postgres.SanctionRepo, runs *postgres.FetchRunRepo, p portal) {
	t.Helper()
	if _, err := usecase.NewLoadSanctions(p, repo, runs, discardObjects{}, time.Now).Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSanctionsAccumulateAndLinkTheCitedCNPJ(t *testing.T) {
	_, db := newServerFor(t, semedHomologacao2025)
	ctx := context.Background()
	repo, runs := postgres.NewSanctionRepo(db), postgres.NewFetchRunRepo(db)
	day1, day2 := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)

	loadSanctions(t, repo, runs, portal{day: day1, ceis: [][]string{ceisSanction("1", fpVieira), ceisSanction("2", "14180324000244")}})
	loadSanctions(t, repo, runs, portal{day: day2, ceis: [][]string{ceisSanction("2", "14180324000244")}})

	sanctions, err := repo.SanctionsByCNPJ(ctx, fpVieira)
	if err != nil || len(sanctions) != 2 {
		t.Fatalf("sanções: %+v %v", sanctions, err)
	}
	seen := map[string]string{}
	for _, s := range sanctions {
		seen[s.Code] = s.FirstSeen.Format(time.DateOnly) + " " + s.LastSeen.Format(time.DateOnly)
	}
	if seen["1"] != "2026-09-24 2026-09-24" || seen["2"] != "2026-09-24 2026-09-25" {
		t.Errorf("primeira e última vez: %+v", seen)
	}
	listed, err := repo.SanctionsListedOn(ctx)
	if err != nil || !listed[domain.RegisterCEIS].Equal(day2) {
		t.Errorf("último arquivo: %+v %v", listed, err)
	}
	certainty := map[string]string{}
	rows, err := db.Query(`SELECT l.record_id, l.certainty FROM entity_links l JOIN entities e ON e.id = l.entity_id
		WHERE l.source = 'cgu_sancoes' AND e.key = $1`, fpVieira)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, c string
		if err := rows.Scan(&id, &c); err != nil {
			t.Fatal(err)
		}
		certainty[id] = c
	}
	if certainty["CEIS:1"] != "exata" || certainty["CEIS:2"] != "forte" {
		t.Errorf("ligações: %+v", certainty)
	}
}

func TestCEPIMImpedimentIsStoredAndLinked(t *testing.T) {
	_, db := newServerFor(t, semedHomologacao2025)
	ctx := context.Background()
	repo := postgres.NewSanctionRepo(db)
	day := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)

	loadSanctions(t, repo, postgres.NewFetchRunRepo(db), portal{day: day,
		cepim: [][]string{{fpVieira, "F P VIEIRA", "633984", "Ministério do Turismo", "INSTAURACAO DE TOMADA DE CONTAS ESPECIAL"}}})

	sanctions, err := repo.SanctionsByCNPJ(ctx, fpVieira)
	if err != nil || len(sanctions) != 1 || sanctions[0].Register != domain.RegisterCEPIM || sanctions[0].Code != fpVieira+"/633984" {
		t.Fatalf("impedimentos: %+v %v", sanctions, err)
	}
	var links int
	if err := db.QueryRow(`SELECT count(*) FROM entity_links WHERE source = 'cgu_sancoes' AND record_id = $1`, "CEPIM:"+fpVieira+"/633984").Scan(&links); err != nil || links != 1 {
		t.Errorf("ligações: %d %v", links, err)
	}
}

func TestSanctionsShowInTheCompanyPageAndMCP(t *testing.T) {
	srv, db := newServerFor(t, semedHomologacao2025)
	day := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	loadSanctions(t, postgres.NewSanctionRepo(db), postgres.NewFetchRunRepo(db), portal{day: day, ceis: [][]string{ceisSanction("1", "14180324000244")}})

	var company struct {
		Sanctions []struct {
			Register string  `json:"register"`
			CNPJ     string  `json:"cnpj"`
			EndsAt   *string `json:"ends_at"`
			State    string  `json:"state"`
		} `json:"sanctions"`
		ListedOn map[string]string `json:"sanctions_listed_on"`
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/14180324000163", &company)
	if len(company.Sanctions) != 1 || company.Sanctions[0].CNPJ != "14180324000244" || company.Sanctions[0].State != "no_cadastro" ||
		company.Sanctions[0].EndsAt == nil || *company.Sanctions[0].EndsAt != "2027-01-01" || company.ListedOn["CEIS"] != "2026-09-25" {
		t.Fatalf("sanções na página da empresa: %+v", company)
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
		Sanctions []struct {
			Cadastro string `json:"cadastro"`
			Estado   string `json:"estado"`
		} `json:"sancoes_cgu"`
		On map[string]string `json:"sancoes_cgu_consultadas_em"`
	}](t, session, "entidade", map[string]any{"numero": "14.180.324/0001-63"})
	if len(out.Sanctions) != 1 || out.Sanctions[0].Cadastro != "CEIS" || out.Sanctions[0].Estado != "no_cadastro" || out.On["CEIS"] != "2026-09-25" {
		t.Errorf("sanções no MCP: %+v", out)
	}
}
