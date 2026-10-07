package http

import (
	"net/http"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type typeTotalDTO struct {
	Type       string `json:"type"`
	Acts       int    `json:"acts"`
	ValueCents int64  `json:"value_cents"`
}

type latestEditionDTO struct {
	GazetteID     string         `json:"gazette_id"`
	Source        string         `json:"source"`
	SourceName    string         `json:"source_name"`
	PublishedAt   string         `json:"published_at"`
	EditionNumber string         `json:"edition_number"`
	IsExtra       bool           `json:"is_extra"`
	TotalActs     int            `json:"total_acts"`
	Types         []typeTotalDTO `json:"types"`
}

type suggestionDTO struct {
	Kind  string `json:"kind"`
	Key   string `json:"key"`
	Label string `json:"label"`
	Name  string `json:"name"`
	Acts  int    `json:"acts"`
}

func (a *API) latestGazettes(w http.ResponseWriter, r *http.Request) {
	editions, err := a.Latest.Execute(r.Context())
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	out := make([]latestEditionDTO, 0, len(editions))
	for _, e := range editions {
		out = append(out, toLatestEditionDTO(e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func toLatestEditionDTO(e domain.LatestEdition) latestEditionDTO {
	dto := latestEditionDTO{GazetteID: e.GazetteID, Source: e.Source, SourceName: domain.SourceName(e.Source), PublishedAt: e.Day.Format(time.DateOnly),
		EditionNumber: e.EditionNumber, IsExtra: e.IsExtra, Types: make([]typeTotalDTO, 0, len(e.Types))}
	for _, t := range e.Types {
		dto.TotalActs += t.Acts
		dto.Types = append(dto.Types, typeTotalDTO{Type: string(t.Type), Acts: t.Acts, ValueCents: t.ValueCents})
	}
	return dto
}

func (a *API) suggest(w http.ResponseWriter, r *http.Request) {
	suggestions, err := a.Suggest.Execute(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	out := make([]suggestionDTO, 0, len(suggestions))
	for _, s := range suggestions {
		out = append(out, suggestionDTO{Kind: string(s.Kind), Key: s.Key, Label: s.Label, Name: s.Name, Acts: s.Acts})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}
