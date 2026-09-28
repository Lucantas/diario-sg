//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type billsResponse struct {
	Total     int            `json:"total"`
	ByPhase   map[string]int `json:"by_phase"`
	ThemeRule string         `json:"theme_rule"`
	Items     []struct {
		Process  string `json:"process"`
		Phase    string `json:"phase"`
		DaysIdle int    `json:"days_idle"`
		Laws     []struct {
			Number    string `json:"number"`
			Certainty string `json:"certainty"`
		} `json:"laws"`
	} `json:"items"`
}

type billResponse struct {
	Process string `json:"process"`
	Phase   string `json:"phase"`
	Events  []struct {
		Text string `json:"text"`
	} `json:"events"`
	Opinions []struct {
		Committee string `json:"committee"`
	} `json:"opinions"`
	Laws []struct {
		Number    string `json:"number"`
		Certainty string `json:"certainty"`
	} `json:"laws"`
}

func seedBills(t *testing.T, ctx context.Context, repo *postgres.BillRepo) {
	t.Helper()
	old := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)
	recent := time.Now().UTC().Add(-48 * time.Hour)
	day := func(y int, m time.Month, d int) *time.Time { v := time.Date(y, m, d, 0, 0, 0, 0, time.UTC); return &v }
	stuck := sampleBill(100, 2025, "PROJETO DE LEI", "Ativo", old, "Recebido na Comissão de COMISSÃO DE JUSTIÇA E REDAÇÃO")
	stuck.Summary, stuck.PresentedOn, stuck.Events[0].At = "DISPÕE SOBRE A PODA DE ÁRVORES", day(2025, 1, 5), old
	viaCommittee := sampleBill(101, 2025, "PROJETO DE LEI", "Ativo", recent, "Recebido na Comissão de COMISSÃO DE DEFESA DO MEIO AMBIENTE")
	viaCommittee.Summary, viaCommittee.PresentedOn, viaCommittee.Events[0].At = "INSTITUI O PROGRAMA HORTA NA ESCOLA", day(2025, 6, 1), recent
	law := sampleBill(102, 2019, "PROJETO DE LEI", "Arquivado", old, "Lei nº. 1147/2020 de 05/02/2020 Publicada em 06/02/2020")
	law.Summary, law.LawNumber, law.LawYear, law.DocNumber, law.DocYear, law.PresentedOn = "INSTITUI O PROJETO CAPOEIRA NA ESCOLA", 1147, 2020, 268, 2019, day(2019, 11, 26)
	byAuthor := sampleBill(103, 2019, "PROJETO DE LEI", "Arquivado", old, "Processo Arquivado - Término de mandato")
	byAuthor.Summary, byAuthor.DocNumber, byAuthor.DocYear, byAuthor.PresentedOn = "INSTITUI O CÓDIGO DE PROTEÇÃO AOS ANIMAIS", 133, 2019, day(2019, 5, 1)
	indication := sampleBill(104, 2025, "INDICAÇÃO LEGISLATIVA", "Ativo", recent, "Entrada no Protocolo Geral")
	indication.Summary = "INDICO A PODA DE ÁRVORE NA RUA X"
	opinions := []domain.BillOpinion{{Position: 1, Result: "Aprovado", Committee: "COMISSÃO DE JUSTIÇA E REDAÇÃO", Rapporteur: "MISAEL"}}
	stuck.Opinions = opinions
	if err := repo.SaveBills(ctx, []domain.Bill{stuck, viaCommittee, law, byAuthor, indication}); err != nil {
		t.Fatal(err)
	}
}

func TestBillsListFiltersByThemePhaseAndIdleDaysAndLinksLaws(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)
	ctx := context.Background()
	seedBills(t, ctx, postgres.NewBillRepo(db))
	norms := []domain.Norm{
		{Kind: domain.NormLaw, Number: 1147, Year: 2020, Summary: "INSTITUI O PROJETO CAPOEIRA NA ESCOLA"},
		{Kind: domain.NormLaw, Number: 1131, Year: 2020, Author: "PROJETO DE LEI Nº 0133/2019 VEREADOR PROFESSOR JOSEMAR", Summary: "INSTITUI O CÓDIGO MUNICIPAL DE PROTEÇÃO AOS ANIMAIS"},
	}
	if err := postgres.NewNormRepo(db).ReplaceNorms(ctx, norms); err != nil {
		t.Fatal(err)
	}

	var theme billsResponse
	getJSON(t, srv.URL+"/v1/bills?theme=meio_ambiente", &theme)
	processes := map[string]string{}
	for _, it := range theme.Items {
		processes[it.Process] = it.Phase
	}
	if theme.Total != 3 || processes["100/2025"] != "em_comissao" || processes["101/2025"] != "em_comissao" || processes["103/2019"] != "arquivado" || theme.ThemeRule == "" {
		t.Fatalf("tema (sem a indicação, que não é normativa): %+v", theme)
	}
	if theme.ByPhase["em_comissao"] != 2 || theme.ByPhase["arquivado"] != 1 {
		t.Fatalf("por fase: %v", theme.ByPhase)
	}

	var stuck billsResponse
	getJSON(t, srv.URL+"/v1/bills?theme=meio_ambiente&phase=em_comissao&min_idle_days=90", &stuck)
	if stuck.Total != 1 || stuck.Items[0].Process != "100/2025" || stuck.Items[0].DaysIdle < 90 {
		t.Fatalf("parados há 90 dias: %+v", stuck)
	}

	var all billsResponse
	getJSON(t, srv.URL+"/v1/bills?q=poda&kind=todos", &all)
	if all.Total != 2 {
		t.Fatalf("todos os tipos: %+v", all)
	}

	var laws billsResponse
	getJSON(t, srv.URL+"/v1/bills?phase=virou_lei", &laws)
	if laws.Total != 1 || len(laws.Items[0].Laws) != 1 || laws.Items[0].Laws[0].Number != "1147/2020" || laws.Items[0].Laws[0].Certainty != "exata" {
		t.Fatalf("lei pelo selo: %+v", laws)
	}

	var byAuthor billResponse
	getJSON(t, srv.URL+"/v1/bills/103-2019", &byAuthor)
	if len(byAuthor.Laws) != 1 || byAuthor.Laws[0].Number != "1131/2020" || byAuthor.Laws[0].Certainty != "forte" || len(byAuthor.Events) != 1 {
		t.Fatalf("lei pelo autor da norma: %+v", byAuthor)
	}

	var one billResponse
	getJSON(t, srv.URL+"/v1/bills/100-2025", &one)
	if one.Phase != "em_comissao" || len(one.Opinions) != 1 || one.Opinions[0].Committee != "COMISSÃO DE JUSTIÇA E REDAÇÃO" {
		t.Fatalf("processo: %+v", one)
	}

	for _, bad := range []string{"/v1/bills?phase=sancionado", "/v1/bills?theme=saude", "/v1/bills/abc"} {
		r, err := http.Get(srv.URL + bad)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		if r.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: %d", bad, r.StatusCode)
		}
	}
	r, _ := http.Get(srv.URL + "/v1/bills/999-2025")
	r.Body.Close()
	if r.StatusCode != http.StatusNotFound {
		t.Errorf("processo inexistente: %d", r.StatusCode)
	}
}
