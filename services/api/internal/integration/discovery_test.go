//go:build integration

package integration

import (
	"database/sql"
	"net/http"
	"net/url"
	"testing"
	"time"
)

const discoveryGazette = "EXTRATO DO CONTRATO Nº 12/2025\nCONTRATADA: Empresa de Limpeza Ltda, CNPJ 12.345.678/0001-90.\n" +
	"PROCESSO: 8.189/2025. OBJETO: limpeza das escolas. VALOR GLOBAL: R$ 50.000,00.\n" +
	"PORTARIA Nº 10/2026\nNomeia FULANO DE TAL para o cargo de assessor.\n" +
	"PORTARIA Nº 11/2026\nNomeia BELTRANO DE TAL para o cargo de assessor."

type suggestions struct {
	Items []struct {
		Kind  string `json:"kind"`
		Key   string `json:"key"`
		Label string `json:"label"`
		Name  string `json:"name"`
		Acts  int    `json:"acts"`
	} `json:"items"`
}

type latestEditions struct {
	Items []struct {
		GazetteID     string `json:"gazette_id"`
		Source        string `json:"source"`
		PublishedAt   string `json:"published_at"`
		EditionNumber string `json:"edition_number"`
		TotalActs     int    `json:"total_acts"`
		Types         []struct {
			Type       string `json:"type"`
			Acts       int    `json:"acts"`
			ValueCents int64  `json:"value_cents"`
		} `json:"types"`
	} `json:"items"`
}

func seedCompany(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO rf_companies (cnpj_base, name, legal_nature, capital_cents, size, reference_month)
		VALUES ('12345678', 'EMPRESA DE LIMPEZA E CONSERVAÇÃO LTDA', '', 0, '', '2026-09-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO rf_establishments (cnpj, cnpj_base, headquarters, trade_name, status, status_reason,
		main_activity_code, main_activity, other_activities, street, number, complement, district, zip, city, uf, reference_month)
		VALUES ('12345678000190', '12345678', true, 'LIMPA TUDO', 'ATIVA', '', '', '', '[]', '', '', '', '', '', '', '', '2026-09-01')`); err != nil {
		t.Fatal(err)
	}
}

func suggest(t *testing.T, base, q string) suggestions {
	t.Helper()
	var out suggestions
	getJSON(t, base+"/v1/suggest?q="+url.QueryEscape(q), &out)
	return out
}

func TestSuggestRecognizesCitedCompaniesAndNumbers(t *testing.T) {
	srv, db := newServerFor(t, discoveryGazette)
	seedCompany(t, db)

	byCNPJ := suggest(t, srv.URL, "12.345.678/0001-90")
	if len(byCNPJ.Items) != 1 || byCNPJ.Items[0].Name != "EMPRESA DE LIMPEZA E CONSERVAÇÃO LTDA" || byCNPJ.Items[0].Acts != 1 {
		t.Errorf("CNPJ citado traz a razão social e os atos: %+v", byCNPJ)
	}
	byName := suggest(t, srv.URL, "conservacao")
	if len(byName.Items) != 1 || byName.Items[0].Key != "12345678000190" || byName.Items[0].Label != "12.345.678/0001-90" {
		t.Errorf("nome sem acento acha a empresa: %+v", byName)
	}
	if got := suggest(t, srv.URL, "limpa tudo"); len(got.Items) != 1 {
		t.Errorf("nome fantasia também acha a empresa: %+v", got)
	}
	byProcess := suggest(t, srv.URL, "processo 8.189/2025")
	if len(byProcess.Items) != 1 || byProcess.Items[0].Kind != "processo" || byProcess.Items[0].Label != "8.189/2025" {
		t.Errorf("processo citado vira sugestão: %+v", byProcess)
	}
	byContract := suggest(t, srv.URL, "12/2025")
	if len(byContract.Items) != 1 || byContract.Items[0].Kind != "contrato" || byContract.Items[0].Key != "12/2025" {
		t.Errorf("só o contrato 12/2025 existe; o processo 122025 fica de fora: %+v", byContract)
	}
	for _, q := range []string{"99.999.999/0001-99", "100%", "ab"} {
		if got := suggest(t, srv.URL, q); len(got.Items) != 0 {
			t.Errorf("%q não sugere nada: %+v", q, got)
		}
	}
}

func TestLatestGazettesSummarizesEachEditionOfTheLastDay(t *testing.T) {
	srv, db := newServerFor(t, discoveryGazette)
	indexAt(t, db, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), "DECRETO Nº 1/2026\nDispõe sobre o horário.")

	var got latestEditions
	getJSON(t, srv.URL+"/v1/gazettes/latest", &got)

	if len(got.Items) != 1 || got.Items[0].PublishedAt != "2026-09-18" || got.Items[0].EditionNumber != "9" || got.Items[0].TotalActs != 3 {
		t.Fatalf("só a edição do último dia entra, com o total de atos: %+v", got)
	}
	types := got.Items[0].Types
	if len(types) != 2 || types[0].Type != "nomeacao" || types[0].Acts != 2 || types[1].Type != "contrato" || types[1].ValueCents != 5000000 {
		t.Errorf("nomeações primeiro; contrato com a soma do valor: %+v", types)
	}

	var inEdition organHits
	getJSON(t, srv.URL+"/v1/acts?type=nomeacao&gazette="+got.Items[0].GazetteID, &inEdition)
	if inEdition.Total != 2 {
		t.Errorf("a busca filtra pela edição: %+v", inEdition)
	}
	resp, body := fetch(t, srv.URL+"/v1/acts?gazette=1412")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("id de edição que não é UUID é filtro inválido: %d %s", resp.StatusCode, body)
	}
}
