package http

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/presentation/ratelimit"
)

const (
	registrationsPerClient   = 100
	registrationsPerInstance = 2000
	grantsPerClient          = 10
	grantsPerInstance        = 200
	tokensPerClient          = 60
	tokensPerInstance        = 600
	protectedResourcePath    = "/.well-known/oauth-protected-resource"
	authServerMetadataPath   = "/.well-known/oauth-authorization-server"
)

type protectedResourceDTO struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
	ResourceName           string   `json:"resource_name"`
}

type authServerDTO struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	AuthorizationResponseIssParameter bool     `json:"authorization_response_iss_parameter_supported"`
}

type registrationRequest struct {
	RedirectURIs []string `json:"redirect_uris"`
	ClientName   string   `json:"client_name"`
}

type registrationDTO struct {
	ClientID                string   `json:"client_id"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientName              string   `json:"client_name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

type tokenDTO struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type oauthErrorDTO struct {
	Error       string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

func (a *API) oauthRoutes(mux *http.ServeMux) {
	if a.OAuth == nil {
		return
	}
	mux.Handle("GET "+protectedResourcePath, openCORS(http.HandlerFunc(a.protectedResource)))
	mux.Handle("GET "+protectedResourcePath+"/{rest...}", openCORS(http.HandlerFunc(a.protectedResource)))
	mux.Handle("GET "+authServerMetadataPath, openCORS(http.HandlerFunc(a.authServerMetadata)))
	mux.Handle("POST /oauth/register", openCORS(a.registerClient(ratelimit.New(registrationsPerClient, registrationsPerInstance, time.Hour, time.Now))))
	mux.Handle("POST /oauth/token", openCORS(a.exchangeToken(ratelimit.New(tokensPerClient, tokensPerInstance, time.Minute, time.Now))))
	mux.HandleFunc("GET /oauth/authorize", a.authorizePage)
	mux.HandleFunc("POST /oauth/authorize", a.authorizeDecision(ratelimit.New(grantsPerClient, grantsPerInstance, time.Hour, time.Now)))
	for _, path := range []string{protectedResourcePath, protectedResourcePath + "/{rest...}", authServerMetadataPath, "/oauth/register", "/oauth/token"} {
		mux.Handle("OPTIONS "+path, openCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })))
	}
}

func (a *API) issuer() string { return strings.TrimRight(a.PublicWebURL, "/") }

func (a *API) oauthEndpoint(name string) string { return a.issuer() + "/api/oauth/" + name }

func (a *API) protectedResource(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, protectedResourceDTO{Resource: a.OAuth.Resource(), AuthorizationServers: []string{a.issuer()},
		BearerMethodsSupported: []string{"header"}, ResourceName: "Diário SG"})
}

func (a *API) authServerMetadata(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, authServerDTO{Issuer: a.issuer(), AuthorizationEndpoint: a.oauthEndpoint("authorize"),
		TokenEndpoint: a.oauthEndpoint("token"), RegistrationEndpoint: a.oauthEndpoint("register"),
		ResponseTypesSupported: []string{"code"}, GrantTypesSupported: []string{"authorization_code"},
		CodeChallengeMethodsSupported: []string{domain.PKCEMethodS256}, TokenEndpointAuthMethodsSupported: []string{"none"},
		AuthorizationResponseIssParameter: true})
}

func (a *API) registerClient(limiter *ratelimit.Limiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(clientKey(r, a.TrustedProxies)) {
			writeJSON(w, http.StatusTooManyRequests, oauthErrorDTO{Error: "slow_down", Description: "muitos registros daqui; tente de novo em uma hora"})
			return
		}
		var req registrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, oauthErrorDTO{Error: domain.OAuthInvalidClientMetadata, Description: "corpo JSON inválido"})
			return
		}
		c, err := a.OAuth.Register(r.Context(), req.ClientName, req.RedirectURIs)
		if err != nil {
			a.writeOAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, registrationDTO{ClientID: c.ID, ClientIDIssuedAt: c.CreatedAt.Unix(), ClientName: c.Name,
			RedirectURIs: c.RedirectURIs, GrantTypes: []string{"authorization_code"}, ResponseTypes: []string{"code"}, TokenEndpointAuthMethod: "none"})
	})
}

func (a *API) exchangeToken(limiter *ratelimit.Limiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !limiter.Allow(clientKey(r, a.TrustedProxies)) {
			writeJSON(w, http.StatusTooManyRequests, oauthErrorDTO{Error: "slow_down", Description: "muitas trocas seguidas; tente de novo em um minuto"})
			return
		}
		if err := r.ParseForm(); err != nil {
			writeJSON(w, http.StatusBadRequest, oauthErrorDTO{Error: domain.OAuthInvalidRequest, Description: "corpo application/x-www-form-urlencoded inválido"})
			return
		}
		token, err := a.OAuth.Exchange(r.Context(), domain.TokenRequest{GrantType: r.PostForm.Get("grant_type"), Code: r.PostForm.Get("code"),
			RedirectURI: r.PostForm.Get("redirect_uri"), ClientID: r.PostForm.Get("client_id"), Verifier: r.PostForm.Get("code_verifier"),
			Resource: r.PostForm.Get("resource")})
		if err != nil {
			a.writeOAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, tokenDTO{AccessToken: token, TokenType: "Bearer"})
	})
}

func (a *API) writeOAuthError(w http.ResponseWriter, err error) {
	var oauthErr *domain.OAuthError
	if errors.As(err, &oauthErr) {
		writeJSON(w, http.StatusBadRequest, oauthErrorDTO{Error: oauthErr.Code, Description: oauthErr.Description})
		return
	}
	a.Log.Error("oauth", "error", err)
	writeJSON(w, http.StatusInternalServerError, oauthErrorDTO{Error: "server_error"})
}

func authorizationRequest(v url.Values) domain.AuthorizationRequest {
	return domain.AuthorizationRequest{ClientID: v.Get("client_id"), RedirectURI: v.Get("redirect_uri"), ResponseType: v.Get("response_type"),
		State: v.Get("state"), CodeChallenge: v.Get("code_challenge"), CodeChallengeMethod: v.Get("code_challenge_method"), Resource: v.Get("resource")}
}

func (a *API) authorizePage(w http.ResponseWriter, r *http.Request) {
	req := authorizationRequest(r.URL.Query())
	c, err := a.OAuth.CheckAuthorization(r.Context(), req)
	if a.handledAuthorizationError(w, err) {
		return
	}
	a.renderConsent(w, http.StatusOK, consentView{Action: a.oauthEndpoint("authorize"), ClientName: c.DisplayName(), ReturnHost: returnTarget(req.RedirectURI),
		WebURL: a.issuer(), Fields: authorizationFields(req)})
}

func (a *API) authorizeDecision(limiter *ratelimit.Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
			a.renderConsent(w, http.StatusForbidden, consentView{WebURL: a.issuer(), Problem: "A autorização só vale clicando na página do Diário SG."})
			return
		}
		if err := r.ParseForm(); err != nil {
			a.renderConsent(w, http.StatusBadRequest, consentView{WebURL: a.issuer(), Problem: "Pedido de autorização inválido."})
			return
		}
		req := authorizationRequest(r.PostForm)
		if r.PostForm.Get("decision") != "allow" {
			_, err := a.OAuth.CheckAuthorization(r.Context(), req)
			if a.handledAuthorizationError(w, err) {
				return
			}
			a.redirectBack(w, r, req, url.Values{"error": {domain.OAuthAccessDenied}})
			return
		}
		if !limiter.Allow(clientKey(r, a.TrustedProxies)) {
			a.renderConsent(w, http.StatusTooManyRequests, consentView{WebURL: a.issuer(), Problem: "Muitas autorizações seguidas daqui; tente de novo em uma hora."})
			return
		}
		code, err := a.OAuth.Authorize(r.Context(), req)
		if a.handledAuthorizationError(w, err) {
			return
		}
		a.redirectBack(w, r, req, url.Values{"code": {code}})
	}
}

func (a *API) handledAuthorizationError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var oauthErr *domain.OAuthError
	switch {
	case errors.Is(err, domain.ErrUnknownOAuthClient), errors.Is(err, domain.ErrRedirectNotAllowed):
		a.renderConsent(w, http.StatusBadRequest, consentView{WebURL: a.issuer(), Problem: err.Error() + "."})
	case errors.As(err, &oauthErr):
		a.renderConsent(w, http.StatusBadRequest, consentView{WebURL: a.issuer(), Problem: "O aplicativo mandou um pedido que o Diário SG não aceita (" + oauthErr.Description + ")."})
	default:
		a.Log.Error("oauth", "error", err)
		a.renderConsent(w, http.StatusInternalServerError, consentView{WebURL: a.issuer(), Problem: "Erro interno; tente de novo."})
	}
	return true
}

func (a *API) redirectBack(w http.ResponseWriter, r *http.Request, req domain.AuthorizationRequest, params url.Values) {
	target, err := url.Parse(req.RedirectURI)
	if err != nil {
		a.renderConsent(w, http.StatusBadRequest, consentView{WebURL: a.issuer(), Problem: domain.ErrRedirectNotAllowed.Error() + "."})
		return
	}
	if req.State != "" {
		params.Set("state", req.State)
	}
	params.Set("iss", a.issuer())
	if target.RawQuery != "" {
		target.RawQuery += "&" + params.Encode()
	} else {
		target.RawQuery = params.Encode()
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func returnTarget(redirectURI string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		return u.Host
	}
	return u.Scheme + "://" + u.Host
}

func authorizationFields(req domain.AuthorizationRequest) map[string]string {
	return map[string]string{"client_id": req.ClientID, "redirect_uri": req.RedirectURI, "response_type": req.ResponseType, "state": req.State,
		"code_challenge": req.CodeChallenge, "code_challenge_method": req.CodeChallengeMethod, "resource": req.Resource}
}

func openCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, MCP-Protocol-Version")
		next.ServeHTTP(w, r)
	})
}

type consentView struct {
	Action     string
	ClientName string
	ReturnHost string
	WebURL     string
	Problem    string
	Fields     map[string]string
}

func (a *API) renderConsent(w http.ResponseWriter, status int, v consentView) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := consentTemplate.Execute(w, v); err != nil {
		a.Log.Error("página de autorização", "error", err)
	}
}

var consentTemplate = template.Must(template.New("consent").Parse(`<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Conectar ao Diário SG</title>
<style>
body{margin:0;font-family:"Public Sans",system-ui,-apple-system,"Segoe UI",sans-serif;background:#f6f7f4;color:#17202b;line-height:1.5}
main{max-width:34rem;margin:0 auto;padding:3rem 1rem}
h1{font-size:1.75rem;line-height:1.15;margin:0 0 1rem}
p{margin:0 0 1rem}
ul{margin:0 0 1.5rem;padding-left:1.2rem;color:#5b6573}
.who{background:#fff;border:1px solid #d8dcd5;border-radius:6px;padding:1rem;margin:0 0 1rem}
.who p:last-child{margin:0}
.actions{display:flex;gap:.75rem;flex-wrap:wrap}
button{font:inherit;font-weight:600;padding:.6rem 1.2rem;border-radius:6px;border:1px solid #1e6b52;cursor:pointer}
button[value=allow]{background:#1e6b52;color:#fff}
button[value=deny]{background:transparent;color:#17202b;border-color:#d8dcd5}
button:focus-visible{outline:3px solid #ffe14d;outline-offset:2px}
.problem{color:#a3302a}
a{color:#1e6b52}
</style>
</head>
<body>
<main>
{{if .Problem}}
<h1>Não deu para conectar</h1>
<p class="problem">{{.Problem}}</p>
<p>Volte ao seu assistente e adicione o conector de novo. <a href="{{.WebURL}}/mcp">Como conectar</a>.</p>
{{else}}
<h1>Conectar ao Diário SG</h1>
<div class="who">
<p><strong>{{.ClientName}}</strong> quer uma chave de acesso ao servidor MCP do Diário SG. O nome é o que o próprio aplicativo informou.</p>
<p>Depois de autorizar, você volta para <strong>{{.ReturnHost}}</strong>.</p>
</div>
<p>A chave:</p>
<ul>
<li>só lê os atos publicados no Diário Oficial e os dados públicos já cruzados no site;</li>
<li>não pede nem guarda seu nome, e-mail ou login;</li>
<li>faz até 60 chamadas por minuto e pode ser revogada a qualquer momento.</li>
</ul>
<form method="post" action="{{.Action}}">
{{range $name, $value := .Fields}}<input type="hidden" name="{{$name}}" value="{{$value}}">
{{end}}<div class="actions">
<button type="submit" name="decision" value="allow">Autorizar</button>
<button type="submit" name="decision" value="deny">Cancelar</button>
</div>
</form>
{{end}}
</main>
</body>
</html>
`))
