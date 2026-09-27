package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const oauthResource = "https://site/api/mcp"

type memOAuth struct {
	used    map[string]bool
	clients map[string]domain.OAuthClient
	codes   map[string]domain.OAuthCode
}

func (m *memOAuth) CreateClient(_ context.Context, c domain.OAuthClient, unusedBefore time.Time) error {
	for id, old := range m.clients {
		if !m.used[id] && old.CreatedAt.Before(unusedBefore) {
			delete(m.clients, id)
		}
	}
	m.clients[c.ID] = c
	return nil
}

func (m *memOAuth) MarkClientUsed(_ context.Context, id string, _ time.Time) error {
	m.used[id] = true
	return nil
}

func (m *memOAuth) Client(_ context.Context, id string) (domain.OAuthClient, error) {
	c, ok := m.clients[id]
	if !ok {
		return c, domain.ErrNotFound
	}
	return c, nil
}

func (m *memOAuth) SaveCode(_ context.Context, code domain.OAuthCode, _ time.Time) error {
	m.codes[code.Hash] = code
	return nil
}

func (m *memOAuth) TakeCode(_ context.Context, hash string) (domain.OAuthCode, error) {
	c, ok := m.codes[hash]
	if !ok {
		return c, domain.ErrNotFound
	}
	delete(m.codes, hash)
	return c, nil
}

type oauthFixture struct {
	uc        *OAuth
	keys      *memAPIKeys
	clock     *time.Time
	client    domain.OAuthClient
	verifier  string
	challenge string
}

func newOAuthFixture(t *testing.T) oauthFixture {
	t.Helper()
	clock := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	keys := newMemAPIKeys()
	uc := NewOAuth(&memOAuth{used: map[string]bool{}, clients: map[string]domain.OAuthClient{}, codes: map[string]domain.OAuthCode{}}, NewAPIKeys(keys), oauthResource,
		func() time.Time { return clock })
	client, err := uc.Register(context.Background(), "Claude", []string{"https://claude.ai/api/mcp/auth_callback"})
	if err != nil {
		t.Fatal(err)
	}
	verifier := strings.Repeat("v", 50)
	sum := sha256.Sum256([]byte(verifier))
	return oauthFixture{uc: uc, keys: keys, clock: &clock, client: client, verifier: verifier, challenge: base64.RawURLEncoding.EncodeToString(sum[:])}
}

func (f oauthFixture) request() domain.AuthorizationRequest {
	return domain.AuthorizationRequest{ClientID: f.client.ID, RedirectURI: f.client.RedirectURIs[0], ResponseType: "code", State: "s",
		CodeChallenge: f.challenge, CodeChallengeMethod: domain.PKCEMethodS256, Resource: oauthResource + "/"}
}

func (f oauthFixture) token(code string) domain.TokenRequest {
	return domain.TokenRequest{GrantType: "authorization_code", Code: code, RedirectURI: f.client.RedirectURIs[0], ClientID: f.client.ID,
		Verifier: f.verifier, Resource: oauthResource}
}

func oauthCode(err error) string {
	var e *domain.OAuthError
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func TestOAuthCodeExchangesForAWorkingKeyOnce(t *testing.T) {
	f := newOAuthFixture(t)
	ctx := context.Background()

	code, err := f.uc.Authorize(ctx, f.request())
	if err != nil {
		t.Fatal(err)
	}
	token, err := f.uc.Exchange(ctx, f.token(code))

	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewAPIKeys(f.keys).Authenticate(ctx, token); err != nil {
		t.Errorf("token não é chave válida: %v", err)
	}
	if _, err := f.uc.Exchange(ctx, f.token(code)); oauthCode(err) != domain.OAuthInvalidGrant {
		t.Errorf("código reusado: %v", err)
	}
}

func TestOAuthCheckAuthorizationSeparatesPageErrorsFromRedirectErrors(t *testing.T) {
	f := newOAuthFixture(t)
	ctx := context.Background()
	unknown, otherURI, noPKCE, plain, token, otherResource := f.request(), f.request(), f.request(), f.request(), f.request(), f.request()
	unknown.ClientID = "nope"
	otherURI.RedirectURI = "https://evil.com/cb"
	noPKCE.CodeChallenge = ""
	plain.CodeChallengeMethod = "plain"
	token.ResponseType = "token"
	otherResource.Resource = "https://outro/mcp"

	if _, err := f.uc.CheckAuthorization(ctx, unknown); !errors.Is(err, domain.ErrUnknownOAuthClient) {
		t.Errorf("cliente desconhecido: %v", err)
	}
	if _, err := f.uc.CheckAuthorization(ctx, otherURI); !errors.Is(err, domain.ErrRedirectNotAllowed) {
		t.Errorf("redirect de fora: %v", err)
	}
	for name, c := range map[string]struct {
		req  domain.AuthorizationRequest
		want string
	}{
		"sem PKCE":       {noPKCE, domain.OAuthInvalidRequest},
		"PKCE plain":     {plain, domain.OAuthInvalidRequest},
		"response_type":  {token, domain.OAuthUnsupportedResponse},
		"outro resource": {otherResource, domain.OAuthInvalidTarget},
	} {
		if _, err := f.uc.CheckAuthorization(ctx, c.req); oauthCode(err) != c.want {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestOAuthExchangeRejectsWrongVerifierClientURIAndExpiredCode(t *testing.T) {
	f := newOAuthFixture(t)
	ctx := context.Background()
	other, err := f.uc.Register(ctx, "Outro", []string{"https://outro.com/cb"})
	if err != nil {
		t.Fatal(err)
	}

	for name, change := range map[string]func(*domain.TokenRequest){
		"verificador": func(r *domain.TokenRequest) { r.Verifier = strings.Repeat("w", 50) },
		"cliente":     func(r *domain.TokenRequest) { r.ClientID = other.ID },
		"uri":         func(r *domain.TokenRequest) { r.RedirectURI = "https://claude.ai/outro" },
	} {
		code, err := f.uc.Authorize(ctx, f.request())
		if err != nil {
			t.Fatal(err)
		}
		req := f.token(code)
		change(&req)
		if _, err := f.uc.Exchange(ctx, req); oauthCode(err) != domain.OAuthInvalidGrant {
			t.Errorf("%s: %v", name, err)
		}
	}

	code, _ := f.uc.Authorize(ctx, f.request())
	*f.clock = f.clock.Add(domain.OAuthCodeTTL)
	if _, err := f.uc.Exchange(ctx, f.token(code)); oauthCode(err) != domain.OAuthInvalidGrant {
		t.Errorf("vencido: %v", err)
	}
	if len(f.keys.byHash) != 0 {
		t.Errorf("chave emitida em troca recusada: %d", len(f.keys.byHash))
	}
}

func TestOAuthExchangeValidatesTheRequestShape(t *testing.T) {
	f := newOAuthFixture(t)
	req := f.token("x")
	req.GrantType = "client_credentials"
	if _, err := f.uc.Exchange(context.Background(), req); oauthCode(err) != domain.OAuthUnsupportedGrant {
		t.Errorf("grant: %v", err)
	}
	req = f.token("")
	if _, err := f.uc.Exchange(context.Background(), req); oauthCode(err) != domain.OAuthInvalidRequest {
		t.Errorf("sem código: %v", err)
	}
}

func TestOAuthRegisterDropsClientsThatNeverGotAToken(t *testing.T) {
	f := newOAuthFixture(t)
	ctx := context.Background()
	code, _ := f.uc.Authorize(ctx, f.request())
	if _, err := f.uc.Exchange(ctx, f.token(code)); err != nil {
		t.Fatal(err)
	}
	idle, _ := f.uc.Register(ctx, "Parado", []string{"https://parado.com/cb"})
	*f.clock = f.clock.Add(domain.UnusedOAuthClientTTL + time.Hour)

	if _, err := f.uc.Register(ctx, "Novo", []string{"https://novo.com/cb"}); err != nil {
		t.Fatal(err)
	}

	if _, err := f.uc.CheckAuthorization(ctx, domain.AuthorizationRequest{ClientID: idle.ID, RedirectURI: "https://parado.com/cb"}); !errors.Is(err, domain.ErrUnknownOAuthClient) {
		t.Errorf("cliente sem uso ficou: %v", err)
	}
	req := f.request()
	req.ResponseType = ""
	if _, err := f.uc.CheckAuthorization(ctx, req); errors.Is(err, domain.ErrUnknownOAuthClient) {
		t.Error("cliente usado foi apagado")
	}
}

func TestOAuthExchangeRejectsAResourceDifferentFromTheAuthorizedOne(t *testing.T) {
	f := newOAuthFixture(t)
	ctx := context.Background()
	req := f.request()
	req.Resource = oauthResource
	code, _ := f.uc.Authorize(ctx, req)
	token := f.token(code)
	token.Resource = "https://outro/api/mcp"

	if _, err := f.uc.Exchange(ctx, token); oauthCode(err) != domain.OAuthInvalidTarget {
		t.Errorf("resource trocado: %v", err)
	}
}
