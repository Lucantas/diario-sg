//go:build integration

package integration

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/email"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const entityAlertGazette = "EXTRATO DO CONTRATO Nº 012/2024\nContratada: Empresa Exemplo LTDA, CNPJ 12.345.678/0001-90. Objeto: merenda escolar.\n" +
	"EXTRATO DO CONTRATO Nº 13/2024\nObjeto: limpeza de escolas."

func TestEntitySubscriptionIsStoredAndListed(t *testing.T) {
	_, db := newServerFor(t, entityAlertGazette)
	ctx := context.Background()
	repo := postgres.NewSubscriptionRepo(db)
	ref := domain.EntityRef{Kind: domain.EntityCNPJ, Key: "12345678000190", Label: "12.345.678/0001-90"}
	s, err := domain.NewEntitySubscription("rep@jornal.com", ref, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.Create(ctx, &s); err != nil {
		t.Fatal(err)
	}
	got, err := repo.FindByConfirmToken(ctx, s.ConfirmToken)
	if err != nil {
		t.Fatal(err)
	}
	if got.Query != "" || got.Entity == nil || *got.Entity != ref {
		t.Fatalf("inscrição lida inesperada: %+v", got)
	}
	if err := got.Confirm(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	active, err := repo.ListActive(ctx)
	if err != nil || len(active) != 1 || active[0].Entity == nil || active[0].Entity.Key != ref.Key {
		t.Fatalf("ListActive inesperado: %+v %v", active, err)
	}

	_, err = db.ExecContext(ctx, `INSERT INTO subscriptions (email, query, entity_kind, entity_key, entity_label, status, confirm_token, unsubscribe_token)
		VALUES ('a@b.com', 'merenda', 'cnpj', '12345678000190', 'x', 'pending', 'c1', 'u1')`)
	if err == nil {
		t.Fatal("termo e entidade juntos deveriam violar o CHECK")
	}
}

func TestEntityHitsInGazetteFollowsTheLinks(t *testing.T) {
	_, db := newServerFor(t, entityAlertGazette)
	ctx := context.Background()
	acts := postgres.NewActRepo(db)
	gazetteID := onlyGazetteID(t, db)

	contrato, err := acts.EntityHitsInGazette(ctx, gazetteID, domain.EntityRef{Kind: domain.EntityContrato, Key: "12/2024"})
	if err != nil || len(contrato) != 1 || !strings.Contains(contrato[0].Title, "012/2024") || !strings.Contains(contrato[0].Snippet, "⟦") {
		t.Fatalf("contrato 12/2024 inesperado: %+v %v", contrato, err)
	}
	cnpj, err := acts.EntityHitsInGazette(ctx, gazetteID, domain.EntityRef{Kind: domain.EntityCNPJ, Key: "12345678000190"})
	if err != nil || len(cnpj) != 1 || cnpj[0].ID != contrato[0].ID || !strings.Contains(cnpj[0].Snippet, "⟦12.345.678/0001-90⟧") {
		t.Fatalf("CNPJ inesperado: %+v %v", cnpj, err)
	}
	none, err := acts.EntityHitsInGazette(ctx, gazetteID, domain.EntityRef{Kind: domain.EntityContrato, Key: "99/2024"})
	if err != nil || len(none) != 0 {
		t.Fatalf("contrato inexistente não deveria trazer atos: %+v %v", none, err)
	}
}

func TestEntityAlertIsSentOncePerEdition(t *testing.T) {
	_, db := newServerFor(t, entityAlertGazette)
	ctx := context.Background()
	subs, box := postgres.NewSubscriptionRepo(db), &inbox{}
	notifier := email.NewNotifier(box, "https://web.exemplo")
	s, err := usecase.NewSubscriptions(subs, notifier).SubscribeEntity(ctx, "rep@jornal.com", domain.EntityCNPJ, "12.345.678/0001-90")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecase.NewSubscriptions(subs, notifier).Confirm(ctx, s.ConfirmToken); err != nil {
		t.Fatal(err)
	}
	match := usecase.NewMatchSubscriptions(postgres.NewGazetteRepo(db), postgres.NewActRepo(db), subs, postgres.NewNotificationLog(db), notifier)

	for i := 0; i < 2; i++ {
		if err := match.Execute(ctx, onlyGazetteID(t, db)); err != nil {
			t.Fatal(err)
		}
	}

	if len(box.msgs) != 2 {
		t.Fatalf("esperava 1 confirmação + 1 alerta, veio %d e-mails", len(box.msgs))
	}
	alert := box.msgs[1]
	if alert.Subject != "CNPJ 12.345.678/0001-90 no Diário da Prefeitura de 18/09" || !strings.Contains(alert.HTML, "012/2024") ||
		strings.Contains(alert.HTML, "13/2024") {
		t.Fatalf("alerta inesperado: %q\n%s", alert.Subject, alert.HTML)
	}
}

func onlyGazetteID(t *testing.T, db *sql.DB) string {
	t.Helper()
	var id string
	if err := db.QueryRow(`SELECT id FROM gazettes`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
