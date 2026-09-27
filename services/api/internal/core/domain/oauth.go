package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	OAuthCodeTTL        = 10 * time.Minute
	PKCEMethodS256      = "S256"
	maxRedirectURIs     = 5
	maxClientNameRunes  = 100
	maxRedirectURILen   = 2000
	oauthSecretBytes    = 32
	minPKCEVerifierLen  = 43
	maxPKCEVerifierLen  = 128
	pkceChallengeLength = 43
)

const (
	OAuthInvalidRequest        = "invalid_request"
	OAuthInvalidClient         = "invalid_client"
	OAuthInvalidGrant          = "invalid_grant"
	OAuthInvalidTarget         = "invalid_target"
	OAuthUnsupportedGrant      = "unsupported_grant_type"
	OAuthUnsupportedResponse   = "unsupported_response_type"
	OAuthAccessDenied          = "access_denied"
	OAuthInvalidRedirectURI    = "invalid_redirect_uri"
	OAuthInvalidClientMetadata = "invalid_client_metadata"
)

var (
	ErrUnknownOAuthClient  = errors.New("cliente OAuth desconhecido; registre o conector de novo")
	ErrRedirectNotAllowed  = errors.New("endereço de retorno não registrado para este cliente")
	errInvalidRedirectURIs = &OAuthError{Code: OAuthInvalidRedirectURI, Description: "redirect_uris: de 1 a 5 endereços https, http em localhost ou esquema de aplicativo (cursor://…), sem fragmento"}
)

type OAuthError struct {
	Code        string
	Description string
}

func (e *OAuthError) Error() string { return e.Code + ": " + e.Description }

type OAuthClient struct {
	ID           string
	Name         string
	RedirectURIs []string
	CreatedAt    time.Time
}

const UnusedOAuthClientTTL = 7 * 24 * time.Hour

type OAuthCode struct {
	Hash        string
	ClientID    string
	RedirectURI string
	Challenge   string
	Resource    string
	ExpiresAt   time.Time
}

type AuthorizationRequest struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
}

type TokenRequest struct {
	GrantType   string
	Code        string
	RedirectURI string
	ClientID    string
	Verifier    string
	Resource    string
}

func NewOAuthClient(name string, redirectURIs []string, now time.Time) (OAuthClient, error) {
	if len(redirectURIs) == 0 || len(redirectURIs) > maxRedirectURIs {
		return OAuthClient{}, errInvalidRedirectURIs
	}
	for _, u := range redirectURIs {
		if !ValidRedirectURI(u) {
			return OAuthClient{}, errInvalidRedirectURIs
		}
	}
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > maxClientNameRunes || strings.ContainsAny(name, "\r\n\t") {
		return OAuthClient{}, &OAuthError{Code: OAuthInvalidClientMetadata, Description: "client_name: até 100 caracteres, numa linha"}
	}
	id, err := NewOAuthSecret()
	if err != nil {
		return OAuthClient{}, err
	}
	return OAuthClient{ID: id, Name: name, RedirectURIs: append([]string(nil), redirectURIs...), CreatedAt: now}, nil
}

func ValidRedirectURI(raw string) bool {
	if len(raw) > maxRedirectURILen || strings.Contains(raw, "#") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Scheme == "" || u.Opaque != "" || !isASCII(u.Host) {
		return false
	}
	switch u.Scheme {
	case "https":
		return u.Host != ""
	case "http":
		return isLoopback(u.Hostname())
	}
	return isPrivateUseScheme(u.Scheme)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

var webOnlySchemes = map[string]bool{"javascript": true, "data": true, "file": true, "vbscript": true, "about": true, "blob": true,
	"filesystem": true, "ftp": true, "ws": true, "wss": true, "mailto": true, "tel": true, "intent": true, "search-ms": true, "search": true}

func isPrivateUseScheme(scheme string) bool {
	return !webOnlySchemes[strings.ToLower(scheme)]
}

func (c OAuthClient) Allows(redirectURI string) bool {
	for _, registered := range c.RedirectURIs {
		if registered == redirectURI || sameLoopbackURI(registered, redirectURI) {
			return true
		}
	}
	return false
}

func (c OAuthClient) DisplayName() string {
	if c.Name != "" {
		return c.Name
	}
	return "Cliente sem nome"
}

func sameLoopbackURI(registered, requested string) bool {
	a, errA := url.Parse(registered)
	b, errB := url.Parse(requested)
	if errA != nil || errB != nil || a.Scheme != "http" || b.Scheme != "http" || !isLoopback(a.Hostname()) {
		return false
	}
	return a.Hostname() == b.Hostname() && a.EscapedPath() == b.EscapedPath() && a.RawQuery == b.RawQuery
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func NewOAuthSecret() (string, error) {
	b := make([]byte, oauthSecretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func IsPKCEChallenge(s string) bool {
	if len(s) != pkceChallengeLength {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(s)
	return err == nil
}

func VerifyPKCE(verifier, challenge string) bool {
	if len(verifier) < minPKCEVerifierLen || len(verifier) > maxPKCEVerifierLen || strings.IndexFunc(verifier, notPKCEChar) >= 0 {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	return subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(challenge)) == 1
}

func notPKCEChar(r rune) bool {
	return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune("-._~", r))
}

func SameResource(a, b string) bool {
	return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
}

func HashOAuthCode(code string) string { return HashAPIKey(code) }
