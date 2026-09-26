package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	SourcePoliticalAgents = "agentes_politicos"

	BodyPrefeitura = "prefeitura"
	BodyCamara     = "camara"

	RolePrefeito        = "prefeito"
	RoleVicePrefeito    = "vice_prefeito"
	RoleSecretario      = "secretario"
	RoleProcuradorGeral = "procurador_geral"
	RoleVereador        = "vereador"

	FirstPrefeituraPayMonth = "2010-10"
	FirstCamaraPayYear      = 2017

	firstLegislatureYear = 2013
	legislatureYears     = 4
	camaraAgentRegime    = "Agente Político"
	camaraCouncillorRole = "VEREADOR"
	camaraNoDataMessage  = "NAO HOUVE INFORMACOES DISPONIBILIZADAS"
)

var prefeituraAgentRoles = map[string]string{
	"PREFEITO":                      RolePrefeito,
	"VICE-PREFEITO":                 RoleVicePrefeito,
	"SECRETARIO MUNICIPAL":          RoleSecretario,
	"PROCURADOR GERAL DO MUNICIPIO": RoleProcuradorGeral,
}

var roleOrder = []string{RolePrefeito, RoleVicePrefeito, RoleSecretario, RoleProcuradorGeral, RoleVereador}

var (
	organCodeRe       = regexp.MustCompile(`^\d+\s*-\s*`)
	councillorTitleRe = regexp.MustCompile(`^VEREADORA?\s+`)
)

type AgentPay struct {
	Body          string
	Month         time.Time
	Name          string
	NameKey       string
	Role          string
	Office        string
	GrossCents    int64
	DiscountCents *int64
	NetCents      *int64
}

type Councillor struct {
	Legislature       int
	Name              string
	NameKey           string
	ParliamentaryName string
	Party             string
	Situation         string
}

type AgentMonth struct {
	Month         time.Time
	Office        string
	GrossCents    int64
	DiscountCents *int64
	NetCents      *int64
}

type PoliticalAgent struct {
	Body              string
	Role              string
	Name              string
	NameKey           string
	Offices           []string
	Party             string
	ParliamentaryName string
	Months            []AgentMonth
}

type SubsidyNorm struct {
	Role     string
	FromYear int
	ToYear   int
	Cents    int64
	Norm     string
	Diario   string
	Search   string
}

var SubsidyNorms = []SubsidyNorm{
	{Role: RolePrefeito, FromYear: 2025, ToYear: 2028, Cents: 2381322, Norm: "Lei nº 1.554/2024", Diario: SourceDiarioPrefeitura, Search: `"LEI N.º 1554/2024"`},
	{Role: RoleVicePrefeito, FromYear: 2025, ToYear: 2028, Cents: 1914816, Norm: "Lei nº 1.554/2024", Diario: SourceDiarioPrefeitura, Search: `"LEI N.º 1554/2024"`},
	{Role: RoleSecretario, FromYear: 2025, ToYear: 2028, Cents: 1675464, Norm: "Lei nº 1.554/2024", Diario: SourceDiarioPrefeitura, Search: `"LEI N.º 1554/2024"`},
	{Role: RoleProcuradorGeral, FromYear: 2025, ToYear: 2028, Cents: 1675464, Norm: "Lei nº 1.554/2024", Diario: SourceDiarioPrefeitura, Search: `"LEI N.º 1554/2024"`},
	{Role: RoleVereador, FromYear: 2025, ToYear: 2028, Cents: 2184043, Norm: "Resolução nº 2.156/2024", Diario: SourceDiarioCamara, Search: `"RESOLUÇÃO Nº 2156/2024"`},
}

func AgentNameKey(name string) string { return foldText(name) }

func LegislatureOf(year int) int { return (year-firstLegislatureYear)/legislatureYears + 1 }

func IsAgentRole(role string) bool { return slices.Contains(roleOrder, role) }

type prefeituraPayResponse struct {
	Success bool `json:"success"`
	Data    []struct {
		Name         string  `json:"nome"`
		Function     string  `json:"funcao"`
		Organogram   string  `json:"organograma"`
		Remuneration float64 `json:"remuneracao"`
	} `json:"data"`
}

func ParsePrefeituraPay(body []byte, year, month int) ([]AgentPay, error) {
	var resp prefeituraPayResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("folha da Prefeitura de %02d/%d: %w", month, year, err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("folha da Prefeitura de %02d/%d: resposta sem sucesso", month, year)
	}
	day := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	var out []AgentPay
	for _, r := range resp.Data {
		role, ok := prefeituraAgentRoles[squeezed(r.Function)]
		if !ok {
			continue
		}
		name := squeezed(r.Name)
		out = append(out, AgentPay{Body: BodyPrefeitura, Month: day, Name: name, NameKey: AgentNameKey(name), Role: role,
			Office: organCodeRe.ReplaceAllString(squeezed(r.Organogram), ""), GrossCents: toCents(r.Remuneration)})
	}
	return out, nil
}

func ParseCamaraPay(body []byte) ([]AgentPay, error) {
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		var empty struct {
			Message string `json:"mensagem"`
		}
		if err := json.Unmarshal(body, &empty); err != nil || !strings.Contains(foldText(empty.Message), camaraNoDataMessage) {
			return nil, fmt.Errorf("folha da Câmara em formato inesperado: %.120q", body)
		}
		return nil, nil
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("folha da Câmara: %w", err)
	}
	var out []AgentPay
	for _, r := range rows {
		if text(r, "regime") != camaraAgentRegime || text(r, "cargo") != camaraCouncillorRole {
			continue
		}
		p, err := camaraAgentPay(r)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func camaraAgentPay(r map[string]any) (AgentPay, error) {
	year, errYear := strconv.Atoi(text(r, "ano"))
	month, errMonth := strconv.Atoi(text(r, "mes"))
	name := text(r, "nome")
	if errYear != nil || errMonth != nil || month < 1 || month > 12 || name == "" {
		return AgentPay{}, fmt.Errorf("linha da folha da Câmara sem ano, mês ou nome: %q %q %q", text(r, "ano"), text(r, "mes"), name)
	}
	items := map[string]int64{}
	for i := 1; ; i++ {
		key := fmt.Sprintf("%02d", i)
		label, ok := r["nome_rem"+key]
		if !ok {
			break
		}
		if s, isText := label.(string); isText {
			value, isNumber := r["valor_rem"+key].(float64)
			if !isNumber {
				return AgentPay{}, fmt.Errorf("vereador %s em %02d/%d: %s sem valor numérico", name, month, year, s)
			}
			items[foldText(s)] = toCents(value)
		}
	}
	gross, ok := items["VENCIMENTOS"]
	if !ok {
		return AgentPay{}, fmt.Errorf("vereador %s em %02d/%d sem vencimentos", name, month, year)
	}
	p := AgentPay{Body: BodyCamara, Month: time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC), Name: name,
		NameKey: AgentNameKey(name), Role: RoleVereador, Office: text(r, "secretaria"), GrossCents: gross}
	if v, ok := items["DESCONTOS"]; ok {
		p.DiscountCents = &v
	}
	if v, ok := items["LIQUIDO"]; ok {
		p.NetCents = &v
	}
	return p, nil
}

func text(r map[string]any, key string) string {
	s, _ := r[key].(string)
	return squeezed(s)
}

func toCents(v float64) int64 { return int64(math.Round(v * 100)) }

func ParseCouncillors(body []byte) ([]Councillor, error) {
	var resp struct {
		Params []struct {
			ParliamentaryName string `json:"Nome_Vereador"`
			Name              string `json:"Nome"`
			Party             string `json:"Partido"`
			Legislature       int    `json:"Legislatura"`
			Situation         string `json:"Situacao"`
		} `json:"Parametros"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("vereadores do SICAM: %w", err)
	}
	out := make([]Councillor, 0, len(resp.Params))
	for _, p := range resp.Params {
		name := squeezed(p.Name)
		out = append(out, Councillor{Legislature: p.Legislature, Name: name, NameKey: AgentNameKey(name),
			ParliamentaryName: squeezed(p.ParliamentaryName), Party: squeezed(p.Party), Situation: squeezed(p.Situation)})
	}
	return out, nil
}

func BuildPoliticalAgents(pay []AgentPay, councillors []Councillor) []PoliticalAgent {
	byKey := map[string]*PoliticalAgent{}
	var order []string
	for _, p := range pay {
		key := p.Body + "|" + p.Role + "|" + p.NameKey
		a, ok := byKey[key]
		if !ok {
			a = &PoliticalAgent{Body: p.Body, Role: p.Role, Name: p.Name, NameKey: p.NameKey}
			byKey[key] = a
			order = append(order, key)
		}
		a.Months = append(a.Months, AgentMonth{Month: p.Month, Office: p.Office, GrossCents: p.GrossCents, DiscountCents: p.DiscountCents, NetCents: p.NetCents})
	}
	out := make([]PoliticalAgent, 0, len(order))
	for _, key := range order {
		a := *byKey[key]
		sort.Slice(a.Months, func(i, j int) bool { return a.Months[i].Month.Before(a.Months[j].Month) })
		for _, m := range a.Months {
			if !slices.Contains(a.Offices, m.Office) {
				a.Offices = append(a.Offices, m.Office)
			}
		}
		if a.Role == RoleVereador {
			if c, ok := councillorOf(a, councillors); ok {
				a.Party, a.ParliamentaryName = c.Party, c.ParliamentaryName
			}
		}
		out = append(out, a)
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := slices.Index(roleOrder, out[i].Role), slices.Index(roleOrder, out[j].Role)
		if ri != rj {
			return ri < rj
		}
		li, lj := out[i].LastMonth(), out[j].LastMonth()
		if !li.Equal(lj) {
			return li.After(lj)
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (a PoliticalAgent) LastMonth() time.Time {
	if len(a.Months) == 0 {
		return time.Time{}
	}
	return a.Months[len(a.Months)-1].Month
}

func (a PoliticalAgent) FirstMonth() time.Time {
	if len(a.Months) == 0 {
		return time.Time{}
	}
	return a.Months[0].Month
}

func councillorOf(a PoliticalAgent, councillors []Councillor) (Councillor, bool) {
	legislature := LegislatureOf(a.LastMonth().Year())
	parliamentary := map[string]bool{}
	for _, m := range a.Months {
		parliamentary[foldText(councillorTitleRe.ReplaceAllString(foldText(m.Office), ""))] = true
	}
	var found Councillor
	ok := false
	for _, c := range councillors {
		if c.NameKey != a.NameKey && !parliamentary[foldText(c.ParliamentaryName)] {
			continue
		}
		if !ok || c.Legislature == legislature {
			found, ok = c, true
		}
	}
	return found, ok
}

func MergeAgentPay(rows []AgentPay) []AgentPay {
	index := map[string]int{}
	var out []AgentPay
	for _, r := range rows {
		key := r.Body + "|" + r.Month.Format(time.DateOnly) + "|" + r.NameKey + "|" + r.Role + "|" + r.Office
		i, ok := index[key]
		if !ok {
			index[key] = len(out)
			out = append(out, r)
			continue
		}
		m := out[i]
		m.GrossCents += r.GrossCents
		m.DiscountCents = addOptional(m.DiscountCents, r.DiscountCents)
		m.NetCents = addOptional(m.NetCents, r.NetCents)
		out[i] = m
	}
	return out
}

func addOptional(a, b *int64) *int64 {
	if a == nil && b == nil {
		return nil
	}
	var sum int64
	for _, v := range []*int64{a, b} {
		if v != nil {
			sum += *v
		}
	}
	return &sum
}

type PoliticalAgentLoad struct {
	Body        string
	From        time.Time
	To          time.Time
	Pay         []AgentPay
	Councillors []Councillor
}

type PayCoverage struct {
	Body string
	From time.Time
	To   time.Time
}

type PoliticalAgentsReport struct {
	Agents   []PoliticalAgent
	Norms    []SubsidyNorm
	Coverage []PayCoverage
}

func PayCoverageOf(pay []AgentPay) []PayCoverage {
	var out []PayCoverage
	for _, body := range []string{BodyPrefeitura, BodyCamara} {
		c := PayCoverage{Body: body}
		for _, p := range pay {
			if p.Body != body {
				continue
			}
			if c.From.IsZero() || p.Month.Before(c.From) {
				c.From = p.Month
			}
			if p.Month.After(c.To) {
				c.To = p.Month
			}
		}
		if !c.From.IsZero() {
			out = append(out, c)
		}
	}
	return out
}

func (a PoliticalAgent) Matches(query string) bool {
	q := foldText(query)
	if q == "" {
		return true
	}
	if strings.Contains(a.NameKey, q) || strings.Contains(foldText(a.ParliamentaryName), q) {
		return true
	}
	return slices.ContainsFunc(a.Offices, func(o string) bool { return strings.Contains(foldText(o), q) })
}
