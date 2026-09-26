package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type panelAmountsDTO struct {
	Contracts       int   `json:"contracts"`
	ContractedCents int64 `json:"contracted_cents"`
	RegisteredCents int64 `json:"registered_cents"`
	AmendedCents    int64 `json:"amended_cents"`
	PaidCents       int64 `json:"paid_cents"`
}

type supplierRowDTO struct {
	CNPJ string `json:"cnpj"`
	Name string `json:"name"`
	panelAmountsDTO
	First   string     `json:"first"`
	Last    string     `json:"last"`
	Organs  []string   `json:"organs"`
	Largest *actHitDTO `json:"largest"`
}

type yearTotalDTO struct {
	Year int `json:"year"`
	panelAmountsDTO
}

type organTotalDTO struct {
	Organ     string `json:"organ"`
	OrganName string `json:"organ_name"`
	panelAmountsDTO
}

type supplierPanelResponse struct {
	Source    string `json:"source"`
	Year      *int   `json:"year"`
	Organ     string `json:"organ"`
	OrganName string `json:"organ_name"`
	Suppliers int    `json:"suppliers"`
	panelAmountsDTO
	Items  []supplierRowDTO `json:"items"`
	Years  []yearTotalDTO   `json:"years"`
	Organs []organTotalDTO  `json:"organs"`
}

func (a *API) supplierPanel(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.PanelFilter{Organ: domain.PrincipalOrgan(strings.ToUpper(strings.TrimSpace(q.Get("organ"))))}
	if y := q.Get("year"); y != "" {
		year, err := strconv.Atoi(y)
		if err != nil || year == 0 {
			writeError(w, domain.ErrInvalidFilter, a.Log)
			return
		}
		f.Year = year
	}
	source := domain.SourceOrDefault(q.Get("source"))
	panel, largest, err := a.Panels.Execute(r.Context(), source, f)
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, toSupplierPanelResponse(source, panel, largest))
}

func toSupplierPanelResponse(source string, p domain.SupplierPanel, largest map[string]domain.ActHit) supplierPanelResponse {
	out := supplierPanelResponse{Source: source, Organ: p.Filter.Organ, OrganName: domain.OrganName(p.Filter.Organ), Suppliers: p.Suppliers,
		panelAmountsDTO: panelAmountsDTO{Contracts: p.Contracts, ContractedCents: p.ContractedCents, RegisteredCents: p.RegisteredCents, AmendedCents: p.AmendedCents, PaidCents: p.PaidCents},
		Items:           make([]supplierRowDTO, 0, len(p.Rows)), Years: make([]yearTotalDTO, 0, len(p.Years)), Organs: make([]organTotalDTO, 0, len(p.Organs))}
	if p.Filter.Year != 0 {
		out.Year = &p.Filter.Year
	}
	for _, row := range p.Rows {
		dto := supplierRowDTO{CNPJ: row.CNPJ, Name: row.Name, First: row.First.Format(time.DateOnly), Last: row.Last.Format(time.DateOnly), Organs: row.Organs,
			panelAmountsDTO: panelAmountsDTO{Contracts: row.Contracts, ContractedCents: row.ContractedCents, RegisteredCents: row.RegisteredCents, AmendedCents: row.AmendedCents, PaidCents: row.PaidCents}}
		if dto.Organs == nil {
			dto.Organs = []string{}
		}
		if h, ok := largest[row.LargestActID]; ok {
			hit := toHitDTO(h)
			dto.Largest = &hit
		}
		out.Items = append(out.Items, dto)
	}
	for _, t := range p.Years {
		year, _ := strconv.Atoi(t.Key)
		out.Years = append(out.Years, yearTotalDTO{Year: year, panelAmountsDTO: amountsOf(t)})
	}
	for _, t := range p.Organs {
		out.Organs = append(out.Organs, organTotalDTO{Organ: t.Key, OrganName: domain.OrganName(t.Key), panelAmountsDTO: amountsOf(t)})
	}
	return out
}

func amountsOf(t domain.PanelTotal) panelAmountsDTO {
	return panelAmountsDTO{Contracts: t.Contracts, ContractedCents: t.ContractedCents, RegisteredCents: t.RegisteredCents, AmendedCents: t.AmendedCents, PaidCents: t.PaidCents}
}
