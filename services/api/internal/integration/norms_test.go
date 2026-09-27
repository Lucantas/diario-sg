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

type normsResponse struct {
	Norms []struct {
		Kind          string  `json:"kind"`
		Number        string  `json:"number"`
		Author        string  `json:"author"`
		PromulgatedOn *string `json:"promulgated_on"`
		TextURL       string  `json:"text_url"`
		DiarioSearch  string  `json:"diario_search"`
	} `json:"norms"`
}

func TestNormsByNumberAndBySummary(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)
	day := time.Date(2022, 12, 8, 0, 0, 0, 0, time.UTC)
	norms := []domain.Norm{
		{Kind: domain.NormLaw, Number: 1406, Year: 2022, Author: "VEREADOR JALMIR JUNIOR", Summary: "PATRIMÔNIO CULTURAL E IMATERIAL", PromulgatedOn: &day, TextURL: "u"},
		{Kind: domain.NormDecree, Number: 1406, Year: 2022, Summary: "NOMEIA"},
		{Kind: domain.NormLaw, Number: 57, Year: 1955, Suffix: "A", Summary: "SUBSÍDIOS DOS VEREADORES"},
	}
	if err := postgres.NewNormRepo(db).ReplaceNorms(context.Background(), norms); err != nil {
		t.Fatal(err)
	}

	var byNumber normsResponse
	getJSON(t, srv.URL+"/v1/norms?kind=lei&number=1.406/2022", &byNumber)
	if len(byNumber.Norms) != 1 || byNumber.Norms[0].Author != "VEREADOR JALMIR JUNIOR" || byNumber.Norms[0].PromulgatedOn == nil ||
		*byNumber.Norms[0].PromulgatedOn != "2022-12-08" || byNumber.Norms[0].DiarioSearch != `"1.406/2022"` {
		t.Fatalf("por número: %+v", byNumber.Norms)
	}

	var bySummary normsResponse
	getJSON(t, srv.URL+"/v1/norms?q=subsidio+vereadores", &bySummary)
	if len(bySummary.Norms) != 1 || bySummary.Norms[0].Number != "57/1955 A" {
		t.Fatalf("por ementa: %+v", bySummary.Norms)
	}

	r, err := http.Get(srv.URL + "/v1/norms?kind=portaria&number=1/2020")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("tipo inválido: %d", r.StatusCode)
	}
}
