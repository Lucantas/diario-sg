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
	Source        string   `json:"source"`
	SourceName    string   `json:"source_name"`
	Position      int      `json:"position"`
	Type          string   `json:"type"`
	Title         string   `json:"title"`
	Organ         string   `json:"organ"`
	OrganName     string   `json:"organ_name"`
	Snippet       string   `json:"snippet"`
	EditionNumber string   `json:"edition_number"`
	PublishedAt   string   `json:"published_at"`
	IsExtra       bool     `json:"is_extra"`
	SourceURL     string   `json:"source_url"`
	PageStart     *int     `json:"page_start"`
	PageEnd       *int     `json:"page_end"`
	PDFSHA256     string   `json:"pdf_sha256"`
	CNPJs         []string `json:"cnpjs"`
	ValuesCents   []int64  `json:"values_cents"`
	Warnings      []string `json:"warnings"`
}

type searchResponse struct {
	Items  []actHitDTO `json:"items"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type actDTO struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Position  int      `json:"position"`
	PageStart *int     `json:"page_start"`
	PageEnd   *int     `json:"page_end"`
	Organ     string   `json:"organ"`
	OrganName string   `json:"organ_name"`
	Warnings  []string `json:"warnings"`
}

type gazetteDTO struct {
	ID            string   `json:"id"`
	Source        string   `json:"source"`
	SourceName    string   `json:"source_name"`
	EditionNumber string   `json:"edition_number"`
	PublishedAt   string   `json:"published_at"`
	IsExtra       bool     `json:"is_extra"`
	SourceURL     string   `json:"source_url"`
	PDFSHA256     string   `json:"pdf_sha256"`
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

type organDTO struct {
	Acronym string `json:"acronym"`
	Name    string `json:"name"`
	Acts    int    `json:"acts"`
}

type organsResponse struct {
	Items []organDTO `json:"items"`
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
	values := h.ValuesCents
	if values == nil {
		values = []int64{}
	}
	return actHitDTO{
		ID: h.ID, GazetteID: h.GazetteID, Source: domain.SourceOrDefault(h.Source), SourceName: domain.SourceName(h.Source), Position: h.Position, Type: string(h.Type), Title: h.Title, Snippet: h.Snippet,
		Organ: h.Organ, OrganName: domain.OrganName(h.Organ),
		EditionNumber: h.EditionNumber, PublishedAt: h.PublishedAt.Format(time.DateOnly), IsExtra: h.IsExtra, SourceURL: h.SourceURL, CNPJs: cnpjs,
		PageStart: pageOrNil(h.PageStart), PageEnd: pageOrNil(h.PageEnd), PDFSHA256: h.Checksum, ValuesCents: values,
		Warnings: domain.ActWarnings(h.WarningFacts()),
	}
}

func toActDTO(a domain.Act) actDTO {
	return actDTO{ID: a.ID, Type: string(a.Type), Title: a.Title, Body: a.Body, Position: a.Position,
		PageStart: pageOrNil(a.PageStart), PageEnd: pageOrNil(a.PageEnd), Organ: a.Organ, OrganName: domain.OrganName(a.Organ),
		Warnings: domain.ActWarnings(domain.WarningFactsOf(a))}
}

func pageOrNil(p int) *int {
	if p <= 0 {
		return nil
	}
	return &p
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
		errors.Is(err, domain.ErrInvalidCNPJ), errors.Is(err, domain.ErrInvalidReport):
		status, msg = http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		status, msg = http.StatusUnauthorized, err.Error()
	case errors.Is(err, domain.ErrSubscriptionCancelled):
		status, msg = http.StatusConflict, err.Error()
	default:
		log.Error("erro não tratado", "error", err)
	}
	writeJSON(w, status, map[string]string{"error": msg})
}
