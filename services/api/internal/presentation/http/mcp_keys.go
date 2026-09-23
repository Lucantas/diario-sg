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
			secret, _ := domain.NewAPIKeySecret()
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

func (a *API) revokeKey(w http.ResponseWriter, r *http.Request) {
	secret, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if err := a.Keys.Revoke(r.Context(), strings.TrimSpace(secret)); err != nil {
		writeError(w, err, a.Log)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
