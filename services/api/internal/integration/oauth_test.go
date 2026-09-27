//go:build integration

package integration

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const oauthCallback = "https://claude.ai/api/mcp/auth_callback"

var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func getJSONStatus(t *testing.T, u string, out any) {
	t.Helper()
	r, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK || r.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("%s: %d %v", u, r.StatusCode, r.Header)
	}
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		t.Fatal(err)
	}
}

func registerOAuthClient(t *testing.T, base string) string {
	t.Helper()
	r, err := http.Post(base+"/oauth/register", "application/json",
		strings.NewReader(`{"client_name":"Claude","redirect_uris":["`+oauthCallback+`"],"token_endpoint_auth_method":"client_secret_basic"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var out struct {
		ClientID   string `json:"client_id"`
		AuthMethod string `json:"token_endpoint_auth_method"`
	}
	_ = json.NewDecoder(r.Body).Decode(&out)
	if r.StatusCode != http.StatusCreated || out.ClientID == "" || out.AuthMethod != "none" {
		t.Fatalf("registro: %d %+v", r.StatusCode, out)
	}
	return out.ClientID
}

func authorizeParams(clientID, challenge string) url.Values {
	return url.Values{"client_id": {clientID}, "redirect_uri": {oauthCallback}, "response_type": {"code"}, "state": {"xyz"},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"}, "resource": {"https://web.exemplo/api/mcp"}}
}

func decide(t *testing.T, base string, params url.Values, decision string) *url.URL {
	t.Helper()
	form := url.Values{}
	for k, v := range params {
		form[k] = v
	}
	form.Set("decision", decision)
	r, err := noRedirect.PostForm(base+"/oauth/authorize", form)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	loc, err := url.Parse(r.Header.Get("Location"))
	if r.StatusCode != http.StatusFound || err != nil || !strings.HasPrefix(loc.String(), oauthCallback+"?") {
		t.Fatalf("decisão %s: %d %q", decision, r.StatusCode, r.Header.Get("Location"))
	}
	if loc.Query().Get("state") != "xyz" || loc.Query().Get("iss") != "https://web.exemplo" {
		t.Errorf("retorno sem state ou iss: %s", loc)
	}
	return loc
}

func TestOAuthFlowGivesAClientAWorkingMCPKey(t *testing.T) {
	srv, _ := newServerFor(t, gazetteText)

	r, err := http.Post(srv.URL+"/mcp", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusUnauthorized ||
		!strings.Contains(r.Header.Get("WWW-Authenticate"), `resource_metadata="https://web.exemplo/.well-known/oauth-protected-resource/api/mcp"`) {
		t.Fatalf("401 sem descoberta: %d %q", r.StatusCode, r.Header.Get("WWW-Authenticate"))
	}
	var resource struct {
		Resource string   `json:"resource"`
		Servers  []string `json:"authorization_servers"`
	}
	getJSONStatus(t, srv.URL+"/.well-known/oauth-protected-resource/api/mcp", &resource)
	if resource.Resource != "https://web.exemplo/api/mcp" || len(resource.Servers) != 1 || resource.Servers[0] != "https://web.exemplo" {
		t.Fatalf("recurso: %+v", resource)
	}
	var meta struct {
		Issuer    string   `json:"issuer"`
		Authorize string   `json:"authorization_endpoint"`
		Token     string   `json:"token_endpoint"`
		Register  string   `json:"registration_endpoint"`
		PKCE      []string `json:"code_challenge_methods_supported"`
	}
	getJSONStatus(t, srv.URL+"/.well-known/oauth-authorization-server", &meta)
	if meta.Issuer != "https://web.exemplo" || meta.Authorize != "https://web.exemplo/api/oauth/authorize" || meta.Token != "https://web.exemplo/api/oauth/token" ||
		meta.Register != "https://web.exemplo/api/oauth/register" || len(meta.PKCE) != 1 || meta.PKCE[0] != "S256" {
		t.Fatalf("metadados: %+v", meta)
	}

	clientID := registerOAuthClient(t, srv.URL)
	verifier := strings.Repeat("abc123", 9)
	sum := sha256.Sum256([]byte(verifier))
	params := authorizeParams(clientID, base64.RawURLEncoding.EncodeToString(sum[:]))

	page, err := http.Get(srv.URL + "/oauth/authorize?" + params.Encode())
	if err != nil {
		t.Fatal(err)
	}
	html, _ := io.ReadAll(page.Body)
	page.Body.Close()
	if page.StatusCode != http.StatusOK || !strings.Contains(string(html), "Claude") || !strings.Contains(string(html), "claude.ai") ||
		!strings.Contains(page.Header.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatalf("consentimento: %d %s", page.StatusCode, html)
	}

	code := decide(t, srv.URL, params, "allow").Query().Get("code")
	token := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {oauthCallback}, "client_id": {clientID},
		"code_verifier": {verifier}, "resource": {"https://web.exemplo/api/mcp"}}
	tr, err := http.PostForm(srv.URL+"/oauth/token", token)
	if err != nil {
		t.Fatal(err)
	}
	var issued struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	_ = json.NewDecoder(tr.Body).Decode(&issued)
	tr.Body.Close()
	if tr.StatusCode != http.StatusOK || issued.TokenType != "Bearer" || !strings.HasPrefix(issued.AccessToken, "dsg_") || tr.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("token: %d %+v", tr.StatusCode, issued)
	}

	session, err := connect(t, srv.URL, issued.AccessToken)
	if err != nil {
		t.Fatalf("MCP com o token: %v", err)
	}
	defer session.Close()
	if _, res := call[struct{}](t, session, "fontes", nil); res.IsError {
		t.Errorf("fontes com o token: %+v", res)
	}

	reuse, err := http.PostForm(srv.URL+"/oauth/token", token)
	if err != nil {
		t.Fatal(err)
	}
	var reuseErr struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(reuse.Body).Decode(&reuseErr)
	reuse.Body.Close()
	if reuse.StatusCode != http.StatusBadRequest || reuseErr.Error != "invalid_grant" {
		t.Errorf("código reusado: %d %+v", reuse.StatusCode, reuseErr)
	}
}

func TestOAuthDenyAndUnknownClient(t *testing.T) {
	srv, _ := newServerFor(t, gazetteText)
	clientID := registerOAuthClient(t, srv.URL)
	params := authorizeParams(clientID, strings.Repeat("A", 43))

	if loc := decide(t, srv.URL, params, "deny"); loc.Query().Get("error") != "access_denied" || loc.Query().Get("code") != "" {
		t.Errorf("cancelar: %s", loc)
	}

	params.Set("client_id", "desconhecido")
	r, err := noRedirect.Get(srv.URL + "/oauth/authorize?" + params.Encode())
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusBadRequest || r.Header.Get("Location") != "" {
		t.Errorf("cliente desconhecido redirecionou: %d %q", r.StatusCode, r.Header.Get("Location"))
	}

	params.Set("client_id", clientID)
	params.Set("code_challenge_method", "plain")
	r, err = noRedirect.Get(srv.URL + "/oauth/authorize?" + params.Encode())
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusBadRequest || r.Header.Get("Location") != "" {
		t.Errorf("PKCE plain virou redirecionamento: %d %q", r.StatusCode, r.Header.Get("Location"))
	}

	params.Set("code_challenge_method", "S256")
	params.Set("decision", "allow")
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/oauth/authorize", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	r, err = noRedirect.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusForbidden || r.Header.Get("Location") != "" {
		t.Errorf("autorização vinda de outro site: %d %q", r.StatusCode, r.Header.Get("Location"))
	}
}

func TestOAuthRedirectKeepsTheRegisteredQuery(t *testing.T) {
	srv, _ := newServerFor(t, gazetteText)
	r, err := http.Post(srv.URL+"/oauth/register", "application/json",
		strings.NewReader(`{"redirect_uris":["https://app.exemplo/cb?z=1&a=2"]}`))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		ClientID string `json:"client_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&out)
	r.Body.Close()
	params := url.Values{"client_id": {out.ClientID}, "redirect_uri": {"https://app.exemplo/cb?z=1&a=2"}, "response_type": {"code"},
		"code_challenge": {strings.Repeat("A", 43)}, "code_challenge_method": {"S256"}, "decision": {"allow"}}

	resp, err := noRedirect.PostForm(srv.URL+"/oauth/authorize", params)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if loc := resp.Header.Get("Location"); !strings.HasPrefix(loc, "https://app.exemplo/cb?z=1&a=2&") || !strings.Contains(loc, "code=") {
		t.Errorf("query do cliente reescrita: %q", loc)
	}
}
