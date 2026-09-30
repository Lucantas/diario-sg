//go:build integration

package integration

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/email"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const filteredAlertGazette = "CONCESSÃO DE LICENÇA MUNICIPAL PRÉVIA\nA Secretaria de Meio Ambiente concede a Licença Municipal Prévia LMP nº 008/2026 à empresa Exemplo Invest, para loteamento no Engenho Pequeno.\n" +
	"EXTRATO DO CONTRATO Nº 012/2024\nContratada: Empresa Exemplo LTDA. Objeto: loteamento de merenda escolar."

func TestFilteredAlertOnlyMatchesTheFilteredActs(t *testing.T) {
	_, db := newServerFor(t, filteredAlertGazette)
	ctx := context.Background()
	subs, box := postgres.NewSubscriptionRepo(db), &inbox{}
	notifier := email.NewNotifier(box, "https://web.exemplo")
	subscriptions := usecase.NewSubscriptions(subs, notifier)
	filter := domain.AlertFilter{Type: domain.ActLicencaAmbiental}
	s, err := subscriptions.Subscribe(ctx, "rep@jornal.com", "loteamento", filter)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := subscriptions.Confirm(ctx, s.ConfirmToken); err != nil {
		t.Fatal(err)
	}
	active, err := subs.ListActive(ctx)
	if err != nil || len(active) != 1 || active[0].Filter != filter {
		t.Fatalf("filtro não voltou do banco: %+v %v", active, err)
	}
	match := usecase.NewMatchSubscriptions(postgres.NewGazetteRepo(db), postgres.NewActRepo(db), subs, postgres.NewNotificationLog(db), notifier)

	if err := match.Execute(ctx, onlyGazetteID(t, db)); err != nil {
		t.Fatal(err)
	}

	if len(box.msgs) != 2 {
		t.Fatalf("esperava 1 confirmação + 1 alerta, veio %d e-mails", len(box.msgs))
	}
	alert := box.msgs[1]
	if !strings.Contains(alert.Subject, "“loteamento” · licença ambiental") || !strings.Contains(alert.HTML, "008/2026") ||
		strings.Contains(alert.HTML, "012/2024") {
		t.Fatalf("alerta inesperado: %q\n%s", alert.Subject, alert.HTML)
	}
}

func TestSubscribeWithFiltersThroughTheAPI(t *testing.T) {
	srv, _ := newServerFor(t, filteredAlertGazette)

	r, body := postWithBody(t, srv.URL+"/v1/subscriptions", `{"email":"rep@jornal.com","filters":{"type":"licenca_ambiental","theme":"meio_ambiente"}}`)
	if r.StatusCode != http.StatusAccepted || !strings.Contains(body, `"subject":"licença ambiental · meio ambiente"`) ||
		!strings.Contains(body, `"theme":"meio_ambiente"`) {
		t.Fatalf("inscrição com filtros inesperada: %d %s", r.StatusCode, body)
	}
	for _, payload := range []string{
		`{"email":"rep@jornal.com"}`,
		`{"email":"rep@jornal.com","filters":{"type":"multa"}}`,
		`{"email":"rep@jornal.com","filters":{"theme":"saude"}}`,
		`{"email":"rep@jornal.com","filters":{"type":"contrato"},"entity":{"kind":"cnpj","value":"12.345.678/0001-90"}}`,
	} {
		if r, body := postWithBody(t, srv.URL+"/v1/subscriptions", payload); r.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s: esperava 400, veio %d %s", payload, r.StatusCode, body)
		}
	}
}
