package events

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/events"
	"github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

type PushHandler struct {
	Index *usecase.IndexGazette
	Match *usecase.MatchSubscriptions
	Runs  *usecase.RecordFetchRun
	Log   *slog.Logger
}

func (h *PushHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("POST /events/gazette-fetched", h.gazetteFetched)
	mux.HandleFunc("POST /events/gazette-indexed", h.gazetteIndexed)
	mux.HandleFunc("POST /events/fetch-completed", h.fetchCompleted)
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

func (h *PushHandler) fetchCompleted(w http.ResponseWriter, r *http.Request) {
	env, ok := h.decode(w, r)
	if !ok {
		return
	}
	var e events.FetchCompleted
	if err := json.Unmarshal(env.Message.Data, &e); err != nil {
		h.ack(w, env, "payload inválido", err)
		return
	}
	run, err := fetchRunOf(e)
	if err != nil {
		h.ack(w, env, "payload inválido", err)
		return
	}
	h.respond(w, env, h.Runs.Execute(r.Context(), run))
}

func fetchRunOf(e events.FetchCompleted) (domain.FetchRun, error) {
	from, err := time.Parse(time.DateOnly, e.RequestedFrom)
	if err != nil {
		return domain.FetchRun{}, err
	}
	to, err := time.Parse(time.DateOnly, e.RequestedTo)
	if err != nil {
		return domain.FetchRun{}, err
	}
	return domain.FetchRun{ID: e.RunID, Source: e.Source, RequestedFrom: from, RequestedTo: to,
		Found: e.Found, Stored: e.Stored, Skipped: e.Skipped, Failed: e.Failed, Error: e.Error,
		StartedAt: e.StartedAt, FinishedAt: e.FinishedAt}, nil
}

func (h *PushHandler) decode(w http.ResponseWriter, r *http.Request) (gcp.PushEnvelope, bool) {
	var env gcp.PushEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		h.Log.Warn("envelope push inválido", "error", err)
		w.WriteHeader(http.StatusNoContent)
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
