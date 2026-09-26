//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

type oversightAPI map[string]string

func (a oversightAPI) Dataset(_ context.Context, name string) ([]byte, error) {
	return []byte(a[name]), nil
}

func oversightFixture() oversightAPI {
	return oversightAPI{
		"prestacao_contas_municipio": `[{"Municipio":"SÃO GONÇALO","Ano":2024,"Indicador":"FAVORÁVEL","Processo":"213638-8/2025","Responsavel":"PREFEITO"},` +
			`{"Municipio":"SÃO GONÇALO","Ano":2025,"Indicador":"EM ANALISE","Processo":"217382-1/2026","Responsavel":"PREFEITO"}]`,
		"penalidades_ressarcimento_municipio": `[{"Processo":"214824-1/2014","AnoCondenacao":2023,"ValorPenalidade":1000.5,"Condenacao":"1/2023-1",` +
			`"Ente":"SAO GONCALO","NomeOrgao":"PREFEITURA SÃO GONÇALO","GrupoNatureza":"TOMADA DE CONTAS","DataSessao":"2023-03-01T00:00:00"},` +
			`{"Processo":"214824-1/2014","AnoCondenacao":2023,"ValorPenalidade":20,"Condenacao":"1/2023-2","Ente":"SAO GONCALO",` +
			`"NomeOrgao":"PREFEITURA SÃO GONÇALO","GrupoNatureza":"TOMADA DE CONTAS","DataSessao":"2023-03-01T00:00:00"}]`,
		"obras_paralisadas": `{"Obras":[{"Ente":"SAO GONCALO","Nome":"PREFEITURA SÃO GONÇALO","NumeroContrato":"025/2016","CNPJContratada":"` + fpVieira +
			`","NomeContratada":"F.P. VIEIRA","ValorTotalContrato":820767.41,"ValorPagoObra":77889.7,"DataParalisacao":"2016-04-01",` +
			`"DataInicioObra":"2015-10-01","MotivoParalisacao":"FATORES VINCULADOS À GESTÃO","StatusContrato":"PRAZO EXPIRADO"}]}`,
	}
}

func loadOversight(t *testing.T, db *sql.DB, api oversightAPI) {
	t.Helper()
	now := func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }
	uc := usecase.NewLoadOversight(api, postgres.NewOversightRepo(db), postgres.NewFetchRunRepo(db), discardObjects{}, now)
	if _, err := uc.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestOversightReplacesTheSetsAndLinksTheContractor(t *testing.T) {
	_, db := newServerFor(t, semedHomologacao2025)

	loadOversight(t, db, oversightFixture())
	loadOversight(t, db, oversightFixture())

	o, err := postgres.NewOversightRepo(db).Oversight(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Accounts) != 2 || o.Accounts[0].Year != 2025 || len(o.Penalties) != 2 || o.Penalties[0].ValueCents+o.Penalties[1].ValueCents != 102050 ||
		len(o.Works) != 1 || o.Works[0].PaidCents != 7788970 {
		t.Fatalf("controle: %+v", o)
	}
	var links int
	if err := db.QueryRow(`SELECT count(*) FROM entity_links WHERE source = $1 AND certainty = 'exata'`, domain.SourceOversight).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if links != 1 {
		t.Fatalf("ligações: %d", links)
	}
}

func TestOversightRouteAndCompanyPage(t *testing.T) {
	srv, db := newServerFor(t, semedHomologacao2025)
	loadOversight(t, db, oversightFixture())

	var o struct {
		Accounts []struct {
			Year    int    `json:"year"`
			Opinion string `json:"opinion"`
		} `json:"accounts"`
		Penalties []struct {
			Process       string            `json:"process"`
			Search        string            `json:"search"`
			TotalCents    int64             `json:"total_cents"`
			Condemnations []json.RawMessage `json:"condemnations"`
		} `json:"penalties"`
		Works []struct {
			CNPJ string `json:"cnpj"`
		} `json:"works"`
	}
	getJSON(t, srv.URL+"/v1/tce", &o)
	if len(o.Accounts) != 2 || o.Accounts[0].Year != 2025 || len(o.Penalties) != 1 || o.Penalties[0].Search != `"214.824" TCE` ||
		o.Penalties[0].TotalCents != 102050 || len(o.Penalties[0].Condemnations) != 2 || len(o.Works) != 1 {
		t.Fatalf("TCE: %+v", o)
	}

	var company struct {
		Works []struct {
			Contract  string `json:"contract"`
			PaidCents int64  `json:"paid_cents"`
			StalledAt string `json:"stalled_at"`
		} `json:"stalled_works"`
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/"+fpVieira, &company)
	if len(company.Works) != 1 || company.Works[0].Contract != "025/2016" || company.Works[0].StalledAt != "2016-04-01" {
		t.Fatalf("obras da empresa: %+v", company.Works)
	}
}
