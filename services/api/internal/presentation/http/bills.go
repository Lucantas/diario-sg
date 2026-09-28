package http

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

type billLawDTO struct {
	Kind         domain.NormKind  `json:"kind"`
	KindName     string           `json:"kind_name"`
	Number       string           `json:"number"`
	Summary      string           `json:"summary"`
	URL          string           `json:"url"`
	Certainty    domain.Certainty `json:"certainty"`
	DiarioSearch string           `json:"diario_search"`
}

type billDTO struct {
	Process      string       `json:"process"`
	Kind         string       `json:"kind"`
	Document     string       `json:"document"`
	Summary      string       `json:"summary"`
	Authors      string       `json:"authors"`
	PresentedOn  *string      `json:"presented_on"`
	Status       string       `json:"status"`
	Phase        string       `json:"phase"`
	DaysIdle     int          `json:"days_idle"`
	CurrentBody  string       `json:"current_body"`
	LastMovement string       `json:"last_movement"`
	URL          string       `json:"url"`
	FetchedAt    string       `json:"fetched_at"`
	Laws         []billLawDTO `json:"laws"`
}

type billEventDTO struct {
	At     string `json:"at"`
	Label  string `json:"label"`
	Text   string `json:"text"`
	Sector string `json:"sector,omitempty"`
}

type billOpinionDTO struct {
	Result     string  `json:"result"`
	On         *string `json:"on"`
	Committee  string  `json:"committee"`
	Rapporteur string  `json:"rapporteur"`
}

type billDetailDTO struct {
	billDTO
	Events   []billEventDTO   `json:"events"`
	Opinions []billOpinionDTO `json:"opinions"`
}

func toBillDTO(s domain.BillSummary) billDTO {
	b := s.Bill
	out := billDTO{Process: b.Key.String(), Kind: b.Kind, Document: b.DocLabel, Summary: b.Summary, Authors: b.Authors,
		PresentedOn: optionalDate(b.PresentedOn), Status: b.Status, Phase: string(s.Phase), DaysIdle: s.DaysIdle,
		CurrentBody: b.CurrentBody, LastMovement: b.LastMovement, URL: b.URL, FetchedAt: b.FetchedAt.Format(time.RFC3339), Laws: []billLawDTO{}}
	for _, l := range s.Laws {
		out.Laws = append(out.Laws, billLawDTO{Kind: l.Norm.Kind, KindName: l.Norm.Kind.Name(), Number: l.Norm.Label(), Summary: l.Norm.Summary, URL: l.URL,
			Certainty: l.Certainty, DiarioSearch: domain.NormDiarioSearch(l.Norm)})
	}
	return out
}

func intQuery(q url.Values, name string) (int, error) {
	s := q.Get(name)
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%w: %s deve ser um número", domain.ErrInvalidInput, name)
	}
	return n, nil
}

func (a *API) bills(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := usecase.BillQuery{Text: q.Get("q"), Author: q.Get("author"), Kind: q.Get("kind"), Status: q.Get("status"),
		Phase: q.Get("phase"), Theme: q.Get("theme"), From: q.Get("from"), To: q.Get("to")}
	var err error
	for name, dst := range map[string]*int{"min_idle_days": &query.MinIdleDays, "limit": &query.Limit, "offset": &query.Offset} {
		if *dst, err = intQuery(q, name); err != nil {
			writeError(w, err, a.Log)
			return
		}
	}
	page, err := a.Bills.List(r.Context(), query)
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	items := make([]billDTO, len(page.Items))
	for i, s := range page.Items {
		items[i] = toBillDTO(s)
	}
	byPhase := map[string]int{}
	for p, n := range page.ByPhase {
		byPhase[string(p)] = n
	}
	w.Header().Set("Cache-Control", "public, max-age=600")
	writeJSON(w, http.StatusOK, map[string]any{"total": page.Total, "by_phase": byPhase, "theme_rule": page.ThemeRule, "items": items})
}

func (a *API) bill(w http.ResponseWriter, r *http.Request) {
	s, err := a.Bills.One(r.Context(), r.PathValue("process"))
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	out := billDetailDTO{billDTO: toBillDTO(s), Events: []billEventDTO{}, Opinions: []billOpinionDTO{}}
	for _, e := range s.Bill.Events {
		out.Events = append(out.Events, billEventDTO{At: e.At.Format(time.RFC3339), Label: e.Label, Text: e.Text, Sector: e.Sector})
	}
	for _, o := range s.Bill.Opinions {
		out.Opinions = append(out.Opinions, billOpinionDTO{Result: o.Result, On: optionalDate(o.On), Committee: o.Committee, Rapporteur: o.Rapporteur})
	}
	w.Header().Set("Cache-Control", "public, max-age=600")
	writeJSON(w, http.StatusOK, out)
}
