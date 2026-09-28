//go:build integration

package integration

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const themeGazette = "PODER EXECUTIVO\nATOS DO PREFEITO\nDECRETO Nº 1/2025\nDispõe sobre o expediente.\nSEMMATRAN\n" +
	"CONCESSÃO DE LICENÇA\nPOSTO ABREU DOIS LTDA torna público que recebeu da Secretaria Municipal de Meio Ambiente e Transportes – SEMMATRAN, " +
	"a LICENÇA MUNICIPAL DE RECUPERAÇÃO E OPERAÇÃO (LMRO) nº 002/2024.\n" +
	"DESPACHO DA PRESIDENTE\nRECURSOS AO CORIM – 1ª Instância - Sessão 06/01/2025. JULGAMENTO DE MULTA: INDEFERIR o processo 03.00121/2024-5.\n" +
	"SEMMA\n" +
	"PORTARIA Nº 5/2025\nDesigna fiscal de contrato.\n" +
	"SEMED\n" +
	"PORTARIA Nº 10/2025\nDesigna servidor para a arborização da escola municipal.\n" +
	"PORTARIA Nº 11/2025\nDesigna servidor para a secretaria da escola.\n"

func TestThemeFiltersActsNormsAndLinksOriginBill(t *testing.T) {
	srv, db := newServerFor(t, themeGazette)
	ctx := context.Background()

	var hits struct {
		Items []struct {
			Title string `json:"title"`
		} `json:"items"`
	}
	getJSON(t, srv.URL+"/v1/acts?theme=meio_ambiente", &hits)
	var titles []string
	for _, h := range hits.Items {
		titles = append(titles, h.Title)
	}
	sort.Strings(titles)
	if strings.Join(titles, "|") != "CONCESSÃO DE LICENÇA|PORTARIA Nº 5/2025" {
		t.Fatalf("atos do tema: %v", titles)
	}

	norms := []domain.Norm{
		{Kind: domain.NormLaw, Number: 1131, Year: 2020, Author: "PROJETO DE LEI Nº 0133/2019 VEREADOR X", Summary: "INSTITUI O CÓDIGO MUNICIPAL DE PROTEÇÃO AOS ANIMAIS"},
		{Kind: domain.NormLaw, Number: 853, Year: 2018, Summary: "CRIA O SERVIÇO DE INSPEÇÃO MUNICIPAL DE PRODUTOS DE ORIGEM ANIMAL"},
	}
	if err := postgres.NewNormRepo(db).ReplaceNorms(ctx, norms); err != nil {
		t.Fatal(err)
	}
	seedBills(t, ctx, postgres.NewBillRepo(db))
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	type normaOut struct {
		Normas []struct {
			Numero  string `json:"numero"`
			Projeto *struct {
				Processo string `json:"processo"`
				Certeza  string `json:"certeza"`
			} `json:"projeto"`
		} `json:"normas"`
		RegraTema string `json:"regra_tema"`
	}
	got, _ := call[normaOut](t, session, "norma", map[string]any{"tema": "meio_ambiente"})
	if len(got.Normas) != 1 || got.Normas[0].Numero != "1131/2020" || got.RegraTema == "" || got.Normas[0].Projeto == nil ||
		got.Normas[0].Projeto.Processo != "103/2019" || got.Normas[0].Projeto.Certeza != "forte" {
		t.Fatalf("norma com tema: %+v", got)
	}

	type actsOut struct {
		Total int `json:"total"`
	}
	acts, _ := call[actsOut](t, session, "buscar_atos", map[string]any{"tema": "meio_ambiente"})
	if acts.Total != 2 {
		t.Fatalf("buscar_atos com tema: %+v", acts)
	}
}
