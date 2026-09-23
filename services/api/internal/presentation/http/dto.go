package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type actHitDTO struct {
	ID            string   `json:"id"`
	GazetteID     string   `json:"gazette_id"`
	Type          string   `json:"type"`
	Title         string   `json:"title"`
	Snippet       string   `json:"snippet"`
	EditionNumber string   `json:"edition_number"`
	PublishedAt   string   `json:"published_at"`
	IsExtra       bool     `json:"is_extra"`
	SourceURL     string   `json:"source_url"`
	CNPJs         []string `json:"cnpjs"`
}

type searchResponse struct {
	Items  []actHitDTO `json:"items"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type actDTO struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Position int    `json:"position"`
}

type gazetteDTO struct {
	ID            string   `json:"id"`
	EditionNumber string   `json:"edition_number"`
	PublishedAt   string   `json:"published_at"`
	IsExtra       bool     `json:"is_extra"`
	SourceURL     string   `json:"source_url"`
	Acts          []actDTO `json:"acts"`
}

type companyResponse struct {
	CNPJ            string         `json:"cnpj"`
	TotalValueCents int64          `json:"total_value_cents"`
	CountByType     map[string]int `json:"count_by_type"`
	Acts            []actHitDTO    `json:"acts"`
}

type monthCountDTO struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

type statsResponse struct {
	Group string          `json:"group"`
	Items []monthCountDTO `json:"items"`
}

type subscribeRequest struct {
	Email string `json:"email"`
	Query string `json:"query"`
}

type tokenRequest struct {
	Token string `json:"token"`
}

type subscriptionDTO struct {
	Query  string `json:"query"`
	Status string `json:"status"`
}

func toHitDTO(h domain.ActHit) actHitDTO {
	cnpjs := h.CNPJs
	if cnpjs == nil {
		cnpjs = []string{}
	}
	return actHitDTO{
		ID: h.ID, GazetteID: h.GazetteID, Type: string(h.Type), Title: h.Title, Snippet: h.Snippet,
		EditionNumber: h.EditionNumber, PublishedAt: h.PublishedAt.Format(time.DateOnly), IsExtra: h.IsExtra, SourceURL: h.SourceURL, CNPJs: cnpjs,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error, log *slog.Logger) {
	status := http.StatusInternalServerError
	msg := "erro interno"
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status, msg = http.StatusNotFound, err.Error()
	case errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrInvalidQuery),
		errors.Is(err, domain.ErrInvalidFilter), errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidCNPJ):
		status, msg = http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrSubscriptionCancelled):
		status, msg = http.StatusConflict, err.Error()
	default:
		log.Error("erro não tratado", "error", err)
	}
	writeJSON(w, status, map[string]string{"error": msg})
}
