package http

import (
	"net/http"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type normDTO struct {
	Kind          domain.NormKind `json:"kind"`
	Number        string          `json:"number"`
	Year          int             `json:"year"`
	Author        string          `json:"author"`
	Summary       string          `json:"summary"`
	PromulgatedOn *string         `json:"promulgated_on"`
	TextURL       string          `json:"text_url"`
	DiarioSearch  string          `json:"diario_search"`
}

func (a *API) norms(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	norms, err := a.Norms.Execute(r.Context(), q.Get("kind"), q.Get("number"), q.Get("q"))
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	out := make([]normDTO, len(norms))
	for i, n := range norms {
		out[i] = normDTO{Kind: n.Kind, Number: n.Label(), Year: n.Year, Author: n.Author, Summary: n.Summary, TextURL: n.TextURL,
			DiarioSearch: domain.NormDiarioSearch(n)}
		if n.PromulgatedOn != nil {
			day := n.PromulgatedOn.Format(time.DateOnly)
			out[i].PromulgatedOn = &day
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, map[string]any{"norms": out})
}
