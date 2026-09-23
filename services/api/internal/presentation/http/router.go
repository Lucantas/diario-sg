package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

type API struct {
	Search        *usecase.SearchActs
	Gazette       *usecase.GetGazette
	Company       *usecase.GetCompany
	Stats         *usecase.ActStats
	Organs        *usecase.ListOrgans
	PDF           *usecase.GetGazettePDF
	Subscriptions *usecase.Subscriptions
	Log           *slog.Logger
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /v1/acts", a.searchActs)
	mux.HandleFunc("GET /v1/gazettes/{id}", a.getGazette)
	mux.HandleFunc("GET /v1/gazettes/{id}/pdf", a.gazettePDF)
	mux.HandleFunc("GET /v1/entities/cnpj/{cnpj}", a.getCompany)
	mux.HandleFunc("GET /v1/stats/acts", a.actStats)
	mux.HandleFunc("GET /v1/organs", a.listOrgans)
	mux.HandleFunc("POST /v1/subscriptions", a.subscribe)

	mux.HandleFunc("POST /v1/subscriptions/confirm", a.confirm)
	mux.HandleFunc("POST /v1/subscriptions/unsubscribe", a.unsubscribe)
	return withMiddleware(mux, a.Log)
}

func (a *API) searchActs(w http.ResponseWriter, r *http.Request) {
	f, ok := a.filterFromQuery(w, r)
	if !ok {
		return
	}
	res, err := a.Search.Execute(r.Context(), f)
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	out := searchResponse{Items: make([]actHitDTO, 0, len(res.Hits)), Total: res.Total, Limit: res.Limit, Offset: res.Offset}
	for _, h := range res.Hits {
		out.Items = append(out.Items, toHitDTO(h))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getGazette(w http.ResponseWriter, r *http.Request) {
	g, acts, err := a.Gazette.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	out := gazetteDTO{ID: g.ID, EditionNumber: g.EditionNumber, PublishedAt: g.PublishedAt.Format(time.DateOnly), IsExtra: g.IsExtra,
		SourceURL: g.SourceURL, PDFSHA256: g.Checksum, Acts: make([]actDTO, 0, len(acts))}
	for _, act := range acts {
		out.Acts = append(out.Acts, toActDTO(act))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getCompany(w http.ResponseWriter, r *http.Request) {
	report, err := a.Company.Execute(r.Context(), r.PathValue("cnpj"))
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	out := companyResponse{CNPJ: report.CNPJ, TotalValueCents: report.TotalCents,
		CountByType: make(map[string]int, len(report.CountByType)), Acts: make([]actHitDTO, 0, len(report.Acts))}
	for t, n := range report.CountByType {
		out.CountByType[string(t)] = n
	}
	for _, h := range report.Acts {
		out.Acts = append(out.Acts, toHitDTO(h))
	}
	writeJSON(w, http.StatusOK, out)
}

const statsGroupMonth = "month"

func (a *API) actStats(w http.ResponseWriter, r *http.Request) {
	if g := r.URL.Query().Get("group"); g != "" && g != statsGroupMonth {
		writeError(w, domain.ErrInvalidFilter, a.Log)
		return
	}
	f, ok := a.filterFromQuery(w, r)
	if !ok {
		return
	}
	counts, err := a.Stats.Execute(r.Context(), f)
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	out := statsResponse{Group: statsGroupMonth, Items: make([]monthCountDTO, 0, len(counts))}
	for _, c := range counts {
		out.Items = append(out.Items, monthCountDTO{Month: c.Month.Format("2006-01"), Count: c.Count})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) listOrgans(w http.ResponseWriter, r *http.Request) {
	organs, err := a.Organs.Execute(r.Context())
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	out := organsResponse{Items: make([]organDTO, 0, len(organs))}
	for _, o := range organs {
		out.Items = append(out.Items, organDTO{Acronym: o.Acronym, Name: o.Name, Acts: o.Acts})
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, out)
}

func (a *API) filterFromQuery(w http.ResponseWriter, r *http.Request) (domain.ActFilter, bool) {
	q := r.URL.Query()
	f := domain.ActFilter{Query: q.Get("q"), Type: domain.ActType(q.Get("type")), Organ: q.Get("organ")}
	f.Limit, _ = strconv.Atoi(q.Get("limit"))
	f.Offset, _ = strconv.Atoi(q.Get("offset"))
	var err error
	if f.From, err = parseDate(q.Get("from")); err != nil {
		writeError(w, domain.ErrInvalidFilter, a.Log)
		return f, false
	}
	if f.To, err = parseDate(q.Get("to")); err != nil {
		writeError(w, domain.ErrInvalidFilter, a.Log)
		return f, false
	}
	if f.MinCents, err = parseReais(q.Get("min_value")); err != nil {
		writeError(w, err, a.Log)
		return f, false
	}
	if f.MaxCents, err = parseReais(q.Get("max_value")); err != nil {
		writeError(w, err, a.Log)
		return f, false
	}
	return f, true
}

func (a *API) subscribe(w http.ResponseWriter, r *http.Request) {
	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, domain.ErrInvalidInput, a.Log)
		return
	}
	s, err := a.Subscriptions.Subscribe(r.Context(), req.Email, req.Query)
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	writeJSON(w, http.StatusAccepted, subscriptionDTO{Query: s.Query, Status: string(s.Status)})
}

func (a *API) confirm(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeError(w, domain.ErrInvalidInput, a.Log)
		return
	}
	s, err := a.Subscriptions.Confirm(r.Context(), req.Token)
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	writeJSON(w, http.StatusOK, subscriptionDTO{Query: s.Query, Status: string(s.Status)})
}

func (a *API) unsubscribe(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeError(w, domain.ErrInvalidInput, a.Log)
		return
	}
	if err := a.Subscriptions.Unsubscribe(r.Context(), req.Token); err != nil {
		writeError(w, err, a.Log)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(domain.SubscriptionCancelled)})
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.DateOnly, s)
}
