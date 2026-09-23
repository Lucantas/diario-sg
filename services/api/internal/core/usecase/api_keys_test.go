package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type memAPIKeys struct {
	byHash  map[string]domain.APIKey
	revoked map[string]bool
	uses    map[string]int
}

func newMemAPIKeys() *memAPIKeys {
	return &memAPIKeys{byHash: map[string]domain.APIKey{}, revoked: map[string]bool{}, uses: map[string]int{}}
}

func (m *memAPIKeys) Create(_ context.Context, hash, prefix string) (domain.APIKey, error) {
	k := domain.APIKey{ID: "k-" + prefix, Prefix: prefix}
	m.byHash[hash] = k
	return k, nil
}

func (m *memAPIKeys) FindActive(_ context.Context, hash string) (domain.APIKey, error) {
	k, ok := m.byHash[hash]
	if !ok || m.revoked[hash] {
		return domain.APIKey{}, domain.ErrNotFound
	}
	return k, nil
}

func (m *memAPIKeys) Revoke(_ context.Context, hash string) error {
	if _, ok := m.byHash[hash]; !ok || m.revoked[hash] {
		return domain.ErrNotFound
	}
	m.revoked[hash] = true
	return nil
}

func (m *memAPIKeys) List(context.Context) ([]domain.APIKey, error) {
	var out []domain.APIKey
	for _, k := range m.byHash {
		out = append(out, k)
	}
	return out, nil
}

func (m *memAPIKeys) RevokeByPrefix(_ context.Context, prefix string) error {
	for hash, k := range m.byHash {
		if k.Prefix == prefix && !m.revoked[hash] {
			m.revoked[hash] = true
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *memAPIKeys) RecordUse(_ context.Context, keyID, tool string) error {
	m.uses[keyID+"/"+tool]++
	return nil
}

func TestIssueStoresOnlyTheHash(t *testing.T) {
	repo := newMemAPIKeys()
	uc := NewAPIKeys(repo)

	secret, key, err := uc.Issue(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.byHash[secret]; ok {
		t.Fatal("o segredo não pode ser guardado em claro")
	}
	if _, ok := repo.byHash[domain.HashAPIKey(secret)]; !ok || key.Prefix != domain.APIKeyPrefix(secret) {
		t.Fatalf("chave guardada errada: %+v", key)
	}
}

func TestAuthenticateRejectsUnknownAndRevokedKeys(t *testing.T) {
	ctx := context.Background()
	uc := NewAPIKeys(newMemAPIKeys())
	secret, issued, _ := uc.Issue(ctx)

	got, err := uc.Authenticate(ctx, secret)
	if err != nil || got.ID != issued.ID {
		t.Fatalf("chave válida recusada: %+v %v", got, err)
	}
	if _, err := uc.Authenticate(ctx, "dsg_outra"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("chave desconhecida deveria ser ErrUnauthorized, veio %v", err)
	}
	if _, err := uc.Authenticate(ctx, ""); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("chave vazia deveria ser ErrUnauthorized, veio %v", err)
	}
	if err := uc.Revoke(ctx, secret); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Authenticate(ctx, secret); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("chave revogada deveria ser ErrUnauthorized, veio %v", err)
	}
	if err := uc.Revoke(ctx, secret); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("revogar de novo deveria ser ErrUnauthorized, veio %v", err)
	}
}

func TestRevokeByPrefixNeedsTheFullPrefix(t *testing.T) {
	ctx := context.Background()
	uc := NewAPIKeys(newMemAPIKeys())
	secret, key, _ := uc.Issue(ctx)

	if err := uc.RevokeByPrefix(ctx, key.Prefix[:4]); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("prefixo curto deveria ser ErrInvalidInput, veio %v", err)
	}
	if err := uc.RevokeByPrefix(ctx, key.Prefix); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Authenticate(ctx, secret); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("chave revogada pelo prefixo ainda autentica: %v", err)
	}
	if err := uc.RevokeByPrefix(ctx, key.Prefix); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("revogar de novo deveria ser ErrNotFound, veio %v", err)
	}
}
