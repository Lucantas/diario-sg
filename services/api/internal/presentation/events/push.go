// Package events é a camada de apresentação para mensagens assíncronas:
// recebe entregas push do Pub/Sub e chama os casos de uso.
//
// Semântica de resposta:
//   - 2xx: mensagem processada (ou inválida de forma permanente) -> ack
//   - 5xx: falha temporária -> Pub/Sub reentrega com backoff e, após N
//     tentativas, envia para a dead-letter queue.
package events

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/seu-usuario/diario-sg/pkg/events"
	"github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

type PushHandler struct {
	Index *usecase.IndexGazette
	Match *usecase.MatchSubscriptions
	Log   *slog.Logger
}

func (h *PushHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("POST /events/gazette-fetched", h.gazetteFetched)
	mux.HandleFunc("POST /events/gazette-indexed", h.gazetteIndexed)
	return mux
}

func (h *PushHandler) gazetteFetched(w http.ResponseWriter, r *http.Request) {
	env, ok := h.decode(w, r)
	if !ok {
		return
	}
	var e events.GazetteFetched
	if err := json.Unmarshal(env.Message.Data, &e); err != nil {
		h.ack(w, env, "payload inválido", err)
		return
	}
	err := h.Index.Execute(r.Context(), usecase.IndexGazetteInput{
		EditionNumber: e.EditionNumber, PublishedAt: e.PublishedAt, SourceURL: e.SourceURL,
		StoragePath: e.StoragePath, Checksum: e.ChecksumSHA256,
	})
	h.respond(w, env, err)
}

func (h *PushHandler) gazetteIndexed(w http.ResponseWriter, r *http.Request) {
	env, ok := h.decode(w, r)
	if !ok {
		return
	}
	var e events.GazetteIndexed
	if err := json.Unmarshal(env.Message.Data, &e); err != nil || e.GazetteID == "" {
		h.ack(w, env, "payload inválido", err)
		return
	}
	h.respond(w, env, h.Match.Execute(r.Context(), e.GazetteID))
}

func (h *PushHandler) decode(w http.ResponseWriter, r *http.Request) (gcp.PushEnvelope, bool) {
	var env gcp.PushEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		h.Log.Warn("envelope push inválido", "error", err)
		w.WriteHeader(http.StatusNoContent) // não adianta reentregar
		return env, false
	}
	return env, true
}

func (h *PushHandler) respond(w http.ResponseWriter, env gcp.PushEnvelope, err error) {
	switch {
	case err == nil:
		h.Log.Info("mensagem processada", "message_id", env.Message.MessageID)
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrNotFound):
		h.ack(w, env, "erro permanente", err)
	default:
		h.Log.Error("falha temporária, será reentregue", "message_id", env.Message.MessageID, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *PushHandler) ack(w http.ResponseWriter, env gcp.PushEnvelope, reason string, err error) {
	h.Log.Warn(reason+": descartando", "message_id", env.Message.MessageID, "error", err)
	w.WriteHeader(http.StatusNoContent)
}
