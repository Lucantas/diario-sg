package http

import (
	"net/http"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type patternSearchDTO struct {
	Type   string `json:"type"`
	From   string `json:"from"`
	To     string `json:"to"`
	Source string `json:"source"`
}

type findingDTO struct {
	Title  string            `json:"title"`
	Detail string            `json:"detail"`
	Acts   []actHitDTO       `json:"acts"`
	Search *patternSearchDTO `json:"search"`
	Link   *findingLinkDTO   `json:"link"`
}

type findingLinkDTO struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type patternDTO struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Rule     string       `json:"rule"`
	Caveat   string       `json:"caveat"`
	Findings []findingDTO `json:"findings"`
}

func (a *API) listPatterns(w http.ResponseWriter, r *http.Request) {
	reports, acts, err := a.Patterns.Execute(r.Context())
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	items := make([]patternDTO, 0, len(reports))
	for _, rep := range reports {
		items = append(items, toPatternDTO(rep, acts))
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func toPatternDTO(rep domain.PatternReport, acts map[string]domain.ActHit) patternDTO {
	dto := patternDTO{ID: string(rep.Pattern.ID), Title: rep.Pattern.Title, Rule: rep.Pattern.Rule, Caveat: rep.Pattern.Caveat,
		Findings: make([]findingDTO, 0, len(rep.Findings))}
	for _, f := range rep.Findings {
		fd := findingDTO{Title: f.Title, Detail: f.Detail, Acts: []actHitDTO{}}
		for _, id := range f.ActIDs {
			if h, ok := acts[id]; ok {
				fd.Acts = append(fd.Acts, toHitDTO(h))
			}
		}
		if f.Search != nil {
			fd.Search = &patternSearchDTO{Type: string(f.Search.Type), Source: f.Search.Source,
				From: f.Search.From.Format(time.DateOnly), To: f.Search.To.Format(time.DateOnly)}
		}
		if f.Link != nil {
			fd.Link = &findingLinkDTO{Label: f.Link.Label, URL: f.Link.URL}
		}
		dto.Findings = append(dto.Findings, fd)
	}
	return dto
}
