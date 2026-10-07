package http

import (
	"net/http"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type latestEditionDTO struct {
	GazetteID     string `json:"gazette_id"`
	EditionNumber string `json:"edition_number"`
	IsExtra       bool   `json:"is_extra"`
}

type typeTotalDTO struct {
	Type       string `json:"type"`
	Acts       int    `json:"acts"`
	ValueCents int64  `json:"value_cents"`
}

type latestDayDTO struct {
	Source      string             `json:"source"`
	SourceName  string             `json:"source_name"`
	PublishedAt string             `json:"published_at"`
	Editions    []latestEditionDTO `json:"editions"`
	Types       []typeTotalDTO     `json:"types"`
}

type suggestionDTO struct {
	Kind  string `json:"kind"`
	Key   string `json:"key"`
	Label string `json:"label"`
	Name  string `json:"name"`
	Acts  int    `json:"acts"`
}

func (a *API) latestGazettes(w http.ResponseWriter, r *http.Request) {
	days, err := a.Latest.Execute(r.Context())
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	out := make([]latestDayDTO, 0, len(days))
	for _, d := range days {
		out = append(out, toLatestDayDTO(d))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func toLatestDayDTO(d domain.LatestDay) latestDayDTO {
	dto := latestDayDTO{Source: d.Source, SourceName: domain.SourceName(d.Source), PublishedAt: d.Day.Format(time.DateOnly),
		Editions: make([]latestEditionDTO, 0, len(d.Editions)), Types: make([]typeTotalDTO, 0, len(d.Types))}
	for _, e := range d.Editions {
		dto.Editions = append(dto.Editions, latestEditionDTO{GazetteID: e.GazetteID, EditionNumber: e.EditionNumber, IsExtra: e.IsExtra})
	}
	for _, t := range d.Types {
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
