package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidRedirectURI(t *testing.T) {
	cases := map[string]bool{
		"https://claude.ai/api/mcp/auth_callback":               true,
		"https://chatgpt.com/connector_platform_oauth_redirect": true,
		"http://localhost:33418/callback":                       true,
		"http://127.0.0.1:8080/cb":                              true,
		"http://[::1]:9000/cb":                                  true,
		"http://example.com/callback":                           false,
		"https://claude.ai/cb#frag":                             false,
		"javascript:alert(1)":                                   false,
		"cursor://anysphere.cursor-retrieval/oauth/callback":    true,
		"com.example.app://cb":                                  true,
		"myapp://cb":                                            true,
		"data://text/html,x":                                    false,
		"com.example.app:/oauth2redirect":                       true,
		"https://clаude.ai/cb":                                  false,
		"https:///cb":                                           false,
		"intent://x#Intent;end":                                 false,
		"file:///etc/passwd":                                    false,
		"https://user:pass@claude.ai/cb":                        false,
		"/relative":                                             false,
		"https://" + strings.Repeat("a", 2000) + ".com/":        false,
	}
	for uri, want := range cases {
		if got := ValidRedirectURI(uri); got != want {
			t.Errorf("%q: %v, esperava %v", uri, got, want)
		}
	}
}

func TestNewOAuthClientRejectsBadMetadata(t *testing.T) {
	now := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	var oauthErr *OAuthError

	if _, err := NewOAuthClient("x", nil, now); !errors.As(err, &oauthErr) || oauthErr.Code != OAuthInvalidRedirectURI {
		t.Errorf("sem redirect_uris: %v", err)
	}
	if _, err := NewOAuthClient("x", []string{"http://evil.com/cb"}, now); !errors.As(err, &oauthErr) || oauthErr.Code != OAuthInvalidRedirectURI {
		t.Errorf("http fora do loopback: %v", err)
	}
	if _, err := NewOAuthClient(strings.Repeat("a", 101), []string{"https://claude.ai/cb"}, now); !errors.As(err, &oauthErr) || oauthErr.Code != OAuthInvalidClientMetadata {
		t.Errorf("nome longo: %v", err)
	}
	c, err := NewOAuthClient("  Claude  ", []string{"https://claude.ai/cb"}, now)
	if err != nil || c.Name != "Claude" || len(c.ID) != 43 || !c.CreatedAt.Equal(now) {
		t.Errorf("cliente: %+v %v", c, err)
	}
}

func TestOAuthClientAllowsExactURIOrLoopbackOnAnyPort(t *testing.T) {
	c := OAuthClient{RedirectURIs: []string{"https://claude.ai/cb", "http://localhost:1234/callback"}}

	for uri, want := range map[string]bool{
		"https://claude.ai/cb":             true,
		"https://claude.ai/cb/":            false,
		"https://claude.ai/cb?x=1":         false,
		"http://localhost:5555/callback":   true,
		"http://localhost/callback":        true,
		"http://localhost:5555/other":      false,
		"http://127.0.0.1:1234/callback":   false,
		"https://localhost:1234/callback":  false,
		"http://localhost:1234/callback?a": false,
	} {
		if got := c.Allows(uri); got != want {
			t.Errorf("%q: %v, esperava %v", uri, got, want)
		}
	}
}

func TestVerifyPKCE(t *testing.T) {
	verifier := strings.Repeat("a1-._~", 8)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	if !IsPKCEChallenge(challenge) || !VerifyPKCE(verifier, challenge) {
		t.Fatal("verificador certo recusado")
	}
	if VerifyPKCE(verifier+"x", challenge) {
		t.Error("verificador errado aceito")
	}
	if VerifyPKCE("curto", challenge) || VerifyPKCE(strings.Repeat("a", 129), challenge) || VerifyPKCE(strings.Repeat("a", 42)+"!", challenge) {
		t.Error("verificador fora do formato aceito")
	}
	if IsPKCEChallenge("curto") || IsPKCEChallenge(strings.Repeat("+", 43)) {
		t.Error("desafio fora do formato aceito")
	}
}

func TestSameResourceIgnoresTrailingSlash(t *testing.T) {
	if !SameResource("https://site/api/mcp/", "https://site/api/mcp") || SameResource("https://site/api/mcp", "https://outro/api/mcp") {
		t.Error("comparação de recurso")
	}
}
