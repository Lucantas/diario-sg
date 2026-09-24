//go:build integration

package integration

import (
	"database/sql"
	"net/http/httptest"
	"net/url"
	"testing"
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
