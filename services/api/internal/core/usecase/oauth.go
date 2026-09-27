package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type OAuth struct {
	repo     ports.OAuthRepository
	keys     *APIKeys
	resource string
	now      func() time.Time
}

func NewOAuth(repo ports.OAuthRepository, keys *APIKeys, resource string, now func() time.Time) *OAuth {
	return &OAuth{repo: repo, keys: keys, resource: resource, now: now}
}

func (uc *OAuth) Resource() string { return uc.resource }

func (uc *OAuth) Register(ctx context.Context, name string, redirectURIs []string) (domain.OAuthClient, error) {
	now := uc.now()
	c, err := domain.NewOAuthClient(name, redirectURIs, now)
	if err != nil {
		return domain.OAuthClient{}, err
	}
	if err := uc.repo.CreateClient(ctx, c, now.Add(-domain.UnusedOAuthClientTTL)); err != nil {
		return domain.OAuthClient{}, err
	}
	return c, nil
}

func (uc *OAuth) CheckAuthorization(ctx context.Context, req domain.AuthorizationRequest) (domain.OAuthClient, error) {
	if req.ClientID == "" {
		return domain.OAuthClient{}, domain.ErrUnknownOAuthClient
	}
	c, err := uc.repo.Client(ctx, req.ClientID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.OAuthClient{}, domain.ErrUnknownOAuthClient
	}
	if err != nil {
		return domain.OAuthClient{}, err
	}
	if !c.Allows(req.RedirectURI) {
		return domain.OAuthClient{}, domain.ErrRedirectNotAllowed
	}
	switch {
	case req.ResponseType != "code":
		return c, &domain.OAuthError{Code: domain.OAuthUnsupportedResponse, Description: "só response_type=code"}
	case req.CodeChallengeMethod != domain.PKCEMethodS256 || !domain.IsPKCEChallenge(req.CodeChallenge):
		return c, &domain.OAuthError{Code: domain.OAuthInvalidRequest, Description: "PKCE obrigatório: code_challenge com code_challenge_method=S256"}
	case req.Resource != "" && !domain.SameResource(req.Resource, uc.resource):
		return c, &domain.OAuthError{Code: domain.OAuthInvalidTarget, Description: "resource deve ser " + uc.resource}
	}
	return c, nil
}

func (uc *OAuth) Authorize(ctx context.Context, req domain.AuthorizationRequest) (string, error) {
	if _, err := uc.CheckAuthorization(ctx, req); err != nil {
		return "", err
	}
	code, err := domain.NewOAuthSecret()
	if err != nil {
		return "", err
	}
	now := uc.now()
	err = uc.repo.SaveCode(ctx, domain.OAuthCode{Hash: domain.HashOAuthCode(code), ClientID: req.ClientID, RedirectURI: req.RedirectURI,
		Challenge: req.CodeChallenge, Resource: req.Resource, ExpiresAt: now.Add(domain.OAuthCodeTTL)}, now)
	if err != nil {
		return "", err
	}
	return code, nil
}

func (uc *OAuth) Exchange(ctx context.Context, req domain.TokenRequest) (string, error) {
	if req.GrantType != "authorization_code" {
		return "", &domain.OAuthError{Code: domain.OAuthUnsupportedGrant, Description: "só grant_type=authorization_code"}
	}
	if req.Code == "" || req.Verifier == "" || req.ClientID == "" || req.RedirectURI == "" {
		return "", &domain.OAuthError{Code: domain.OAuthInvalidRequest, Description: "faltam code, code_verifier, client_id ou redirect_uri"}
	}
	code, err := uc.repo.TakeCode(ctx, domain.HashOAuthCode(req.Code))
	if errors.Is(err, domain.ErrNotFound) {
		return "", invalidGrant("código desconhecido ou já usado")
	}
	if err != nil {
		return "", err
	}
	switch {
	case !uc.now().Before(code.ExpiresAt):
		return "", invalidGrant("código vencido")
	case code.ClientID != req.ClientID || code.RedirectURI != req.RedirectURI:
		return "", invalidGrant("código de outro cliente ou de outro redirect_uri")
	case !domain.VerifyPKCE(req.Verifier, code.Challenge):
		return "", invalidGrant("code_verifier não confere")
	case req.Resource != "" && (!domain.SameResource(req.Resource, uc.resource) || code.Resource != "" && !domain.SameResource(req.Resource, code.Resource)):
		return "", &domain.OAuthError{Code: domain.OAuthInvalidTarget, Description: "resource deve ser " + uc.resource}
	}
	secret, _, err := uc.keys.Issue(ctx)
	if err != nil {
		return "", err
	}
	if err := uc.repo.MarkClientUsed(ctx, code.ClientID, uc.now()); err != nil {
		return "", err
	}
	return secret, nil
}

func invalidGrant(description string) error {
	return &domain.OAuthError{Code: domain.OAuthInvalidGrant, Description: description}
}
