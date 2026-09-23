//go:build integration

package integration

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func postJSON(t *testing.T, url, body string) int {
	t.Helper()
	r, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	return r.StatusCode
}

func TestErrorReportQueue(t *testing.T) {
	srv, db := newServerFor(t, warningsGazette)
	var hits warnedHits
	getJSON(t, srv.URL+"/v1/acts", &hits)
	gid := hits.Items[0].GazetteID
	ctx := context.Background()
	repo := postgres.NewErrorReportRepo(db)

	cases := []struct {
		body string
		want int
	}{
		{`{"gazette_id":"` + gid + `","position":1,"act_title":"PORTARIA Nº 5/2026","kind":"texto_errado","message":"o texto ficou no ato de cima"}`, http.StatusAccepted},
		{`{"gazette_id":"` + gid + `","position":1,"act_title":"PORTARIA","kind":"texto_errado","website":"http://spam"}`, http.StatusAccepted},
		{`{"gazette_id":"` + gid + `","position":1,"act_title":"PORTARIA","kind":"bobagem"}`, http.StatusBadRequest},
		{`{"gazette_id":"` + gid + `","act_title":"PORTARIA","kind":"tipo_errado"}`, http.StatusBadRequest},
		{`{"gazette_id":"00000000-0000-0000-0000-000000000000","position":1,"act_title":"X","kind":"tipo_errado"}`, http.StatusNotFound},
		{`{"gazette_id":"` + gid + `","position":1,"act_title":"X","kind":"tipo_errado"}`, http.StatusTooManyRequests},
	}
	for _, c := range cases {
		if got := postJSON(t, srv.URL+"/v1/reports", c.body); got != c.want {
			t.Errorf("%s: esperava %d, veio %d", c.body, c.want, got)
		}
	}

	bad := domain.ErrorReport{GazetteID: "nao-e-uuid", ActTitle: "X", Kind: domain.ReportWrongType, Status: domain.ReportOpen}
	if err := repo.Create(ctx, &bad); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("edição com id inválido deveria ser ErrNotFound, veio %v", err)
	}

	open, err := repo.List(ctx, domain.ReportOpen)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].Kind != domain.ReportWrongText || open[0].PageStart != 1 || open[0].EditionNumber != "9" {
		t.Fatalf("fila inesperada: %+v", open)
	}

	if err := repo.Close(ctx, open[0].ID, domain.ReportResolved); err != nil {
		t.Fatal(err)
	}
	if err := repo.Close(ctx, open[0].ID, domain.ReportDiscarded); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("reporte já fechado não fecha de novo: %v", err)
	}
	resolved, err := repo.List(ctx, domain.ReportResolved)
	if err != nil || len(resolved) != 1 || resolved[0].ClosedAt.IsZero() {
		t.Fatalf("resolvidos inesperados: %+v %v", resolved, err)
	}
}
