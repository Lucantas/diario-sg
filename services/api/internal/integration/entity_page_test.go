//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const processGazette = "ATOS DO PREFEITO\nDECRETO Nº 9/2024\nCita o processo nº 2808/2022 sem órgão.\n" +
	"SEMAD\nAVISO DE LICITAÇÃO\nPregão nº 4/2023. Processo Administrativo nº 2808/2022.\n" +
	"HOMOLOGAÇÃO/ADJUDICAÇÃO\nPregão nº 4/2023, Processo nº 2.808/2022. Contratada: Empresa Exemplo LTDA, CNPJ: 12.345.678/0001-90.\n" +
	"EXTRATO DO CONTRATO Nº 30/SEMAD/2023\nProcesso nº 2808/2022. Contratada: Empresa Exemplo LTDA, CNPJ: 12.345.678/0001-90. Valor global: R$ 50.000,00.\n" +
	"SEMTRAN\nEXTRATO DO CONTRATO Nº 7/SEMTRAN/2024\nProcesso nº 2808/2022. Adesão à ata do Pregão nº 4/2023.\n" +
	"SEMAD\nEXTRATO DO PRIMEIRO TERMO ADITIVO AO CONTRATO Nº 30/SEMAD/2023\nProcesso Administrativo nº 2808/2022. Prorrogação.\n"

func newEntityServer(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()
	return newServerFor(t, processGazette)
}

func TestSearchHitsCarryProcessAndContractMentions(t *testing.T) {
	srv, _ := newEntityServer(t)
	var res struct {
		Items []struct {
			Title    string `json:"title"`
			Mentions []struct {
				Kind, Key, Label, Slug string
			} `json:"mentions"`
		} `json:"items"`
	}
	getJSON(t, srv.URL+"/v1/acts?q="+url.QueryEscape("30/SEMAD/2023"), &res)
	var found bool
	for _, it := range res.Items {
		for _, m := range it.Mentions {
			if m.Kind == "contrato" && m.Key == "30/SEMAD/2023" && m.Slug == "30-SEMAD-2023" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("esperava a menção ao contrato 30/SEMAD/2023: %+v", res.Items)
	}
}

func TestSearchHitsDoNotExposePhase(t *testing.T) {
	srv, _ := newEntityServer(t)

	_, body := fetch(t, srv.URL+"/v1/acts?q="+url.QueryEscape("2808/2022"))

	if !strings.Contains(body, `"mentions"`) {
		t.Fatalf("esperava mentions nos itens da busca: %s", body)
	}
	if strings.Contains(body, `"phase"`) {
		t.Fatalf("a busca não deve trazer phase (só o relatório de entidade, na Task 4): %s", body)
	}
}

func TestProcessReportGroupsOrgansPhasesAndRelated(t *testing.T) {
	_, db := newEntityServer(t)

	report, err := usecase.NewGetEntity(postgres.NewLinkRepo(db)).
		Execute(context.Background(), domain.EntityProcesso, "2808-2022", "")

	if err != nil {
		t.Fatal(err)
	}
	if report.Label != "2808/2022" || report.TotalActs != 6 {
		t.Fatalf("rótulo e total: %q %d", report.Label, report.TotalActs)
	}
	wantOrgans := []domain.OrganCount{{Organ: "SEMAD", Acts: 4}, {Organ: "SEMTRAN", Acts: 1}, {Organ: "", Acts: 1}}
	if fmt.Sprint(report.Organs) != fmt.Sprint(wantOrgans) {
		t.Fatalf("órgãos: %+v", report.Organs)
	}
	if report.CountByPhase[domain.PhaseLicitacao] != 1 || report.CountByPhase[domain.PhaseHomologacao] != 1 ||
		report.CountByPhase[domain.PhaseContrato] != 2 || report.CountByPhase[domain.PhaseAditivo] != 1 ||
		report.CountByPhase[domain.PhaseOutro] != 1 {
		t.Fatalf("fases: %v", report.CountByPhase)
	}
	for _, a := range report.Acts {
		if a.Phase == "" {
			t.Fatalf("ato sem fase: %q", a.Title)
		}
	}
	related := map[string]domain.RelatedEntity{}
	for _, r := range report.Related {
		if r.Kind == domain.EntityProcesso {
			t.Fatalf("processo não traz outros processos em citados junto: %+v", r)
		}
		related[string(r.Kind)+":"+r.Key] = r
	}
	contract, cnpj := related["contrato:30/SEMAD/2023"], related["cnpj:12345678000190"]
	if contract.Acts != 2 || contract.Label != "30/SEMAD/2023" || cnpj.Acts != 2 || cnpj.Label != "12.345.678/0001-90" {
		t.Fatalf("citados junto: %+v", report.Related)
	}
	if _, self := related["processo:28082022"]; self {
		t.Fatal("a própria entidade não entra em citados junto")
	}
}

func TestProcessReportReturnsUpToThreeHundredActs(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= 120; i++ {
		fmt.Fprintf(&b, "DESPACHO DO SECRETÁRIO\nProcesso nº 4444/2024. Despacho %d.\n", i)
	}
	_, db := newServerFor(t, b.String())

	report, err := postgres.NewLinkRepo(db).ReportByKey(context.Background(), domain.EntityProcesso, "44442024", "")

	if err != nil || report.TotalActs != 120 || len(report.Acts) != 120 {
		t.Fatalf("esperava os 120 atos: total %d, devolvidos %d, %v", report.TotalActs, len(report.Acts), err)
	}
}

func TestEntityRouteServesProcessAndContract(t *testing.T) {
	srv, _ := newEntityServer(t)
	var p struct {
		Label        string                   `json:"label"`
		TotalActs    int                      `json:"total_acts"`
		CountByPhase map[string]int           `json:"count_by_phase"`
		Organs       []struct{ Organ string } `json:"organs"`
		Warnings     []string                 `json:"warnings"`
		Acts         []struct{ Phase string } `json:"acts"`
	}
	getJSON(t, srv.URL+"/v1/entities/processo/2808-2022", &p)
	if p.Label != "2808/2022" || p.TotalActs != 6 || len(p.Organs) != 3 || len(p.Warnings) != 1 || p.Acts[0].Phase == "" {
		t.Fatalf("processo: %+v", p)
	}
	var c struct {
		Key       string   `json:"key"`
		TotalActs int      `json:"total_acts"`
		Warnings  []string `json:"warnings"`
	}
	getJSON(t, srv.URL+"/v1/entities/contrato/30-SEMAD-2023", &c)
	if c.Key != "30/SEMAD/2023" || c.TotalActs != 2 {
		t.Fatalf("contrato: %+v", c)
	}
	if c.Warnings == nil || len(c.Warnings) != 0 {
		t.Fatalf("contrato sem aviso deve trazer warnings como lista vazia, não null: %+v", c.Warnings)
	}
	for path, status := range map[string]int{
		"/v1/entities/valor/100":           http.StatusNotFound,
		"/v1/entities/processo/12":         http.StatusBadRequest,
		"/v1/entities/processo/99999-2001": http.StatusOK,
	} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != status {
			t.Errorf("%s: esperava %d, veio %d", path, status, res.StatusCode)
		}
	}
}

func TestMCPEntityIncludesPhaseOrgansAndRelated(t *testing.T) {
	srv, _ := newEntityServer(t)
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	type organOut struct {
		Organ     string `json:"orgao"`
		OrganName string `json:"orgao_nome"`
		Acts      int    `json:"atos"`
	}
	type relatedOut struct {
		Kind  string `json:"tipo"`
		Key   string `json:"chave"`
		Label string `json:"rotulo"`
		Acts  int    `json:"atos"`
	}
	type entityOut struct {
		Label        string         `json:"rotulo"`
		CountByPhase map[string]int `json:"atos_por_fase"`
		Organs       []organOut     `json:"orgaos"`
		Related      []relatedOut   `json:"citados_junto"`
		Warning      string         `json:"aviso"`
		Acts         []struct {
			Phase string `json:"fase"`
		} `json:"atos_recentes"`
	}

	process, res := call[entityOut](t, session, "entidade", map[string]any{"tipo": "processo", "numero": "2808/2022"})
	if res.IsError {
		t.Fatalf("entidade processo: %+v", res)
	}
	if process.Label != "2808/2022" || process.CountByPhase["contrato"] != 2 || len(process.Organs) != 3 {
		t.Fatalf("processo: %+v", process)
	}
	var foundContract bool
	for _, r := range process.Related {
		if r.Kind == "contrato" && r.Key == "30/SEMAD/2023" {
			foundContract = true
		}
	}
	if !foundContract {
		t.Fatalf("citados junto sem o contrato: %+v", process.Related)
	}
	if !strings.Contains(process.Warning, "aparece em 2 órgãos") {
		t.Fatalf("aviso: %q", process.Warning)
	}
	if len(process.Acts) == 0 || process.Acts[0].Phase == "" {
		t.Fatalf("ato recente sem fase: %+v", process.Acts)
	}

	cnpj, res2 := call[entityOut](t, session, "entidade", map[string]any{"numero": "12.345.678/0001-90"})
	if res2.IsError {
		t.Fatalf("entidade cnpj: %+v", res2)
	}
	if len(cnpj.Organs) == 0 {
		t.Fatalf("cnpj deveria trazer órgãos: %+v", cnpj)
	}
	if len(cnpj.Related) != 0 {
		t.Fatalf("cnpj não deveria trazer citados_junto: %+v", cnpj.Related)
	}
}

func TestRelatedLeavesOutEntitiesOfTheSameKind(t *testing.T) {
	_, db := newServerFor(t, "SEMAD\nEXTRATO DO CONTRATO Nº 30/SEMAD/2023\n"+
		"Processo nº 2808/2022 e Processo nº 1111/2023. Substitui o Contrato nº 5/SEMAD/2022. "+
		"Contratada: Empresa Exemplo LTDA, CNPJ: 12.345.678/0001-90.\n")
	repo := postgres.NewLinkRepo(db)

	for _, c := range []struct {
		kind  domain.EntityKind
		key   string
		wants []domain.EntityKind
	}{
		{domain.EntityProcesso, "28082022", []domain.EntityKind{domain.EntityContrato, domain.EntityCNPJ}},
		{domain.EntityContrato, "30/SEMAD/2023", []domain.EntityKind{domain.EntityProcesso, domain.EntityCNPJ}},
	} {
		report, err := repo.ReportByKey(context.Background(), c.kind, c.key, "")
		if err != nil {
			t.Fatal(err)
		}
		kinds := map[domain.EntityKind]bool{}
		for _, r := range report.Related {
			kinds[r.Kind] = true
		}
		if kinds[c.kind] || !kinds[c.wants[0]] || !kinds[c.wants[1]] {
			t.Fatalf("%s %s: citados junto com tipos errados: %+v", c.kind, c.key, report.Related)
		}
	}
}
