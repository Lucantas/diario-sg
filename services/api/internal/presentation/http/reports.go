package http

import (
	"encoding/json"
	"net/http"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type reportRequest struct {
	GazetteID string `json:"gazette_id"`
	Position  *int   `json:"position"`
	ActTitle  string `json:"act_title"`
	Kind      string `json:"kind"`
	Message   string `json:"message"`
	Website   string `json:"website"`
}

var reportAccepted = map[string]string{"status": "recebido"}

func (a *API) reportError(limiter *rateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.allow(clientKey(r)) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "muitos reportes seguidos; tente de novo em um minuto"})
			return
		}
		var req reportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Position == nil {
			writeError(w, domain.ErrInvalidInput, a.Log)
			return
		}
		if req.Website != "" {
			writeJSON(w, http.StatusAccepted, reportAccepted)
			return
		}
		err := a.Reports.Submit(r.Context(), req.GazetteID, *req.Position, req.ActTitle, domain.ReportKind(req.Kind), req.Message)
		if err != nil {
			writeError(w, err, a.Log)
			return
		}
		writeJSON(w, http.StatusAccepted, reportAccepted)
	}
}
