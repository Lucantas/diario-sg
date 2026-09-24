//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
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
