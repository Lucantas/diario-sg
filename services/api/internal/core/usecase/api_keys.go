package usecase

import (
	"context"
	"errors"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type APIKeys struct{ repo ports.APIKeyRepository }

func NewAPIKeys(r ports.APIKeyRepository) *APIKeys { return &APIKeys{repo: r} }

func (uc *APIKeys) Issue(ctx context.Context) (string, domain.APIKey, error) {
	secret, err := domain.NewAPIKeySecret()
	if err != nil {
		return "", domain.APIKey{}, err
	}
	key, err := uc.repo.Create(ctx, domain.HashAPIKey(secret), domain.APIKeyPrefix(secret))
	if err != nil {
		return "", domain.APIKey{}, err
	}
	return secret, key, nil
}

func (uc *APIKeys) Authenticate(ctx context.Context, secret string) (domain.APIKey, error) {
	if domain.APIKeyPrefix(secret) == "" {
		return domain.APIKey{}, domain.ErrUnauthorized
	}
	key, err := uc.repo.FindActive(ctx, domain.HashAPIKey(secret))
	return key, unauthorizedIfMissing(err)
}

func (uc *APIKeys) Revoke(ctx context.Context, secret string) error {
	if domain.APIKeyPrefix(secret) == "" {
		return domain.ErrUnauthorized
	}
	return unauthorizedIfMissing(uc.repo.Revoke(ctx, domain.HashAPIKey(secret)))
}

func (uc *APIKeys) RecordUse(ctx context.Context, keyID, tool string) error {
	return uc.repo.RecordUse(ctx, keyID, tool)
}

func unauthorizedIfMissing(err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		return domain.ErrUnauthorized
	}
	return err
}
