//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

func TestAPIKeysListAndRevokeByPrefix(t *testing.T) {
	_, db := newServerFor(t, gazetteText)
	ctx := context.Background()
	uc := usecase.NewAPIKeys(postgres.NewAPIKeyRepo(db))
	secret, used, err := uc.Issue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, unused, _ := uc.Issue(ctx)
	for _, tool := range []string{"buscar_atos", "buscar_atos", "fontes"} {
		if err := uc.RecordUse(ctx, used.ID, tool); err != nil {
			t.Fatal(err)
		}
	}

	if err := uc.RevokeByPrefix(ctx, used.Prefix); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Authenticate(ctx, secret); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("chave revogada pelo prefixo ainda autentica: %v", err)
	}
	if err := uc.RevokeByPrefix(ctx, used.Prefix); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("revogar de novo deveria ser ErrNotFound, veio %v", err)
	}

	keys, err := uc.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byPrefix := map[string]domain.APIKey{}
	for _, k := range keys {
		byPrefix[k.Prefix] = k
	}
	got := byPrefix[used.Prefix]
	if len(keys) != 2 || got.RecentCalls != 3 || got.LastUsedAt.Year() <= 1970 || got.RevokedAt.Year() <= 1970 {
		t.Fatalf("chave usada e revogada listada errado: %+v", got)
	}
	if other := byPrefix[unused.Prefix]; other.RecentCalls != 0 || other.RevokedAt.Year() > 1970 || other.LastUsedAt.Year() > 1970 {
		t.Fatalf("chave sem uso listada errado: %+v", other)
	}
}
