//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const camaraText = "PODER LEGISLATIVO\nCÂMARA MUNICIPAL DE SÃO GONÇALO\nSão Gonçalo, 3 de novembro de 2025\n" +
	"Ano-08 / Edição - 138\n\nDIÁRIO OFICIAL ELETRÔNICO – D.O.E\nLEI MUNICIPAL 855/2018 DE 05/07/2018.\n" +
	"EXTRATO DO CONTRATO Nº 12/2025\nCONTRATADA: Empresa de Limpeza Ltda, CNPJ 12.345.678/0001-90.\n" +
	"PROCESSO: 8.189/2025. OBJETO: limpeza do plenário da Câmara. VALOR: R$ 50.000,00.\nPágina 1 de 1\n"

func indexCamara(t *testing.T, db *sql.DB) {
	t.Helper()
	in := usecase.IndexGazetteInput{PublishedAt: time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC),
		SourceURL:   "https://www.cmsg.rj.gov.br/diariooficialeletronico/PUBLICACOES/2025-11-03.pdf",
		StoragePath: "raw/diario_camara/2025/11/03/2025-11-03.pdf", Checksum: strings.Repeat("c", 64), Source: domain.SourceDiarioCamara}
	idx := usecase.NewIndexGazette(postgres.NewGazetteRepo(db), stringStore(camaraText), passthroughExtractor{}, parser.Set{}, entities.New(), &recPub{})
	if err := idx.Execute(context.Background(), in); err != nil {
		t.Fatal(err)
	}
}

type sourcedHits struct {
	Items []struct {
		Source     string `json:"source"`
		SourceName string `json:"source_name"`
		Title      string `json:"title"`
		GazetteID  string `json:"gazette_id"`
	} `json:"items"`
	Total int `json:"total"`
}

func searchSources(t *testing.T, url string) sourcedHits {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: status %d", url, resp.StatusCode)
	}
	var out sourcedHits
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestCamaraEditionIsIndexedWithItsSource(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)
	indexCamara(t, db)

	var source, edition string
	if err := db.QueryRow(`SELECT source, edition_number FROM gazettes WHERE checksum = $1`, strings.Repeat("c", 64)).Scan(&source, &edition); err != nil {
		t.Fatal(err)
	}
	if source != domain.SourceDiarioCamara || edition != "138" {
		t.Fatalf("edição da Câmara gravada com fonte %q e número %q", source, edition)
	}

	var camaraLinks int
	if err := db.QueryRow(`
		SELECT count(*) FROM entity_links l JOIN acts a ON a.id::text = l.record_id JOIN gazettes g ON g.id = a.gazette_id
		WHERE g.source = 'diario_camara' AND l.source = 'diario_camara'`).Scan(&camaraLinks); err != nil {
		t.Fatal(err)
	}
	if camaraLinks == 0 {
		t.Fatal("as ligações da Câmara deveriam levar a fonte da edição")
	}

	camara := searchSources(t, srv.URL+"/v1/acts?source=diario_camara")
	if camara.Total != 1 || camara.Items[0].Source != domain.SourceDiarioCamara ||
		camara.Items[0].SourceName != "Diário Oficial Eletrônico da Câmara Municipal de São Gonçalo" ||
		camara.Items[0].Title != "EXTRATO DO CONTRATO Nº 12/2025" {
		t.Fatalf("busca na Câmara: %+v", camara)
	}
	all := searchSources(t, srv.URL+"/v1/acts?q=limpeza")
	sources := map[string]bool{}
	for _, it := range all.Items {
		sources[it.Source] = true
	}
	if !sources[domain.SourceDiarioCamara] {
		t.Fatalf("busca sem fonte deveria trazer a Câmara também: %+v", all)
	}

	resp, err := http.Get(srv.URL + "/v1/acts?source=tce")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("fonte desconhecida deveria ser 400, veio %d", resp.StatusCode)
	}
}

func TestEntityReportSpansBothSources(t *testing.T) {
	_, db := newServerFor(t, gazetteText)
	indexCamara(t, db)

	report, err := postgres.NewLinkRepo(db).ReportByKey(context.Background(), domain.EntityCNPJ, "12345678000190", "")
	if err != nil {
		t.Fatal(err)
	}

	sources := map[string]bool{}
	for _, a := range report.Acts {
		sources[a.Source] = true
	}
	if !sources[domain.SourceDiarioCamara] || !sources[domain.SourceDiarioPrefeitura] {
		t.Fatalf("o relatório do CNPJ deveria juntar as duas fontes: %+v", sources)
	}
}

func TestMCPReadsTheCamara(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)
	indexCamara(t, db)
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	found, _ := call[struct {
		Total int `json:"total"`
		Acts  []struct {
			actRef
			Diario string `json:"diario"`
		} `json:"atos"`
	}](t, session, "buscar_atos", map[string]any{"diario": "diario_camara"})
	if found.Total != 1 || found.Acts[0].Diario != "diario_camara" {
		t.Fatalf("busca na Câmara: %+v", found)
	}

	act, _ := call[struct {
		Diario   string `json:"diario"`
		Citation string `json:"citacao"`
	}](t, session, "ler_ato", map[string]any{"edicao_id": found.Acts[0].GazetteID, "posicao": found.Acts[0].Position})
	if act.Diario != "diario_camara" ||
		!strings.HasPrefix(act.Citation, "SÃO GONÇALO (RJ). Câmara Municipal. Diário Oficial Eletrônico da Câmara Municipal de São Gonçalo, ed. 138, 3 nov. 2025") {
		t.Fatalf("ato da Câmara: %+v", act)
	}

	sources, _ := call[struct {
		Sources []struct {
			Diario   string   `json:"diario"`
			From     string   `json:"de"`
			Gazettes int      `json:"edicoes"`
			Gaps     []string `json:"lacunas"`
		} `json:"fontes"`
	}](t, session, "fontes", nil)
	if len(sources.Sources) != 2 || sources.Sources[1].Diario != "diario_camara" || sources.Sources[1].Gazettes != 1 ||
		sources.Sources[1].From != "2025-11-03" || !strings.Contains(strings.Join(sources.Sources[1].Gaps, " "), "2020-10-04") {
		t.Fatalf("fontes: %+v", sources)
	}

	if _, res := call[struct{}](t, session, "buscar_atos", map[string]any{"diario": "tce"}); !res.IsError {
		t.Fatal("diário desconhecido deveria voltar como erro da ferramenta")
	}
}

func TestSameProcessoInBothDiariosIsWeak(t *testing.T) {
	_, db := newServerFor(t, gazetteText)
	indexCamara(t, db)
	links := postgres.NewLinkRepo(db)

	both, err := links.ReportByKey(context.Background(), domain.EntityProcesso, "81892025", "")
	if err != nil {
		t.Fatal(err)
	}
	if both.Sources != 2 || both.Certainty != domain.CertaintyWeak || both.TotalActs != 2 {
		t.Fatalf("processo nos dois diários deveria ser fraco: %+v", both)
	}

	camara, err := links.ReportByKey(context.Background(), domain.EntityProcesso, "81892025", domain.SourceDiarioCamara)
	if err != nil {
		t.Fatal(err)
	}
	if camara.Sources != 1 || camara.Certainty != domain.CertaintyStrong || camara.TotalActs != 1 ||
		len(camara.Acts) != 1 || camara.Acts[0].Source != domain.SourceDiarioCamara || camara.TotalCents != 5000000 {
		t.Fatalf("só a Câmara: %+v", camara)
	}
}
