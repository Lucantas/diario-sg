package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/presentation/ratelimit"
)

type issueKeyRequest struct {
	Website string `json:"website"`
}

type issuedKeyDTO struct {
	Key    string `json:"key"`
	Prefix string `json:"prefix"`
	MCPURL string `json:"mcp_url"`
}

func (a *API) issueKey(limiter *ratelimit.Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(clientKey(r)) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "muitas chaves geradas daqui; tente de novo em uma hora"})
			return
		}
		var req issueKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, domain.ErrInvalidInput, a.Log)
			return
		}
		if req.Website != "" {
			secret, err := domain.NewAPIKeySecret()
			if err != nil {
				writeError(w, err, a.Log)
				return
			}
			writeJSON(w, http.StatusCreated, a.issuedKey(secret))
			return
		}
		secret, _, err := a.Keys.Issue(r.Context())
		if err != nil {
			writeError(w, err, a.Log)
			return
		}
		writeJSON(w, http.StatusCreated, a.issuedKey(secret))
	}
}

func (a *API) issuedKey(secret string) issuedKeyDTO {
	return issuedKeyDTO{Key: secret, Prefix: domain.APIKeyPrefix(secret), MCPURL: strings.TrimRight(a.PublicWebURL, "/") + "/api/mcp"}
}

func (a *API) revokeKey(limiter *ratelimit.Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(clientKey(r)) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "muitas tentativas seguidas; tente de novo em um minuto"})
			return
		}
		if err := a.Keys.Revoke(r.Context(), bearerToken(r)); err != nil {
			writeError(w, err, a.Log)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func bearerToken(r *http.Request) string {
	fields := strings.Fields(r.Header.Get("Authorization"))
	if len(fields) != 2 || !strings.EqualFold(fields[0], "bearer") {
		return ""
	}
	return fields[1]
}

func perClient(limiter *ratelimit.Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(clientKey(r)) {
			w.Header().Set("Retry-After", "60")
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "muitas chamadas seguidas; tente de novo em um minuto"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
