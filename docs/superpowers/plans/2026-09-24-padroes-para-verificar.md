# Padrões para verificar — plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/v1/patterns` e página `/padroes` com duas regras fixas: dispensas do mesmo fornecedor que somadas passam do limite no ano, e picos de nomeação/exoneração antes da eleição municipal.

**Architecture:** o Postgres entrega dispensas candidatas (com corpo e processos) e contagens mensais; funções puras do domínio aplicam as regras; o caso de uso monta relatórios e carrega os atos citados; HTTP serializa; o front mostra.

**Tech Stack:** Go (`database/sql`, `lib/pq`), Postgres, React + Vite + Vitest.

**Spec:** `docs/superpowers/specs/2026-09-24-padroes-para-verificar-design.md`

## Global Constraints

- `AGENTS.md`: sem comentários explicativos no código; commits em português (conventional commits); sem link de sessão.
- Linguagem: "padrão para verificar", nunca "irregularidade".
- Limites (centavos): 8.666 — 800000 até 18/07/2018, 1760000 a partir de 19/07/2018; 14.133 — 5000000 (2021–2022), 5720833 (2023), 5990602 (2024), 6272559 (2025), 6549211 (2026). 14.133 vale a partir de 30/12/2023 para todos e, de 01/04/2021 a 29/12/2023, para ato que cita "14.133".
- Eleições: 07/10/2012, 02/10/2016, 15/11/2020, 06/10/2024. Janela: 6 meses antes do mês da eleição. Razão: 1,5 × mediana.
- Só Diário da Prefeitura (`diario_prefeitura`).
- Verificação: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.

---

### Task 1: regras de texto, limite e formatação (domínio)

**Files:** Create `services/api/internal/core/domain/{dispensa_text.go,dispensa_limit.go,money.go}` e testes `*_test.go`.

**Produces:** `CitesValueDispensa(body string) bool`, `CitesLei14133(body string) bool`, `CitesEmergency(body string) bool`, `IsRepublication(body string) bool`, `DispensaLimitCents(published time.Time, citesLei14133 bool) int64`, `FormatBRL(cents int64) string`, `civilDate(y int, m time.Month, d int) time.Time`.

- [ ] Testes (falham):

```go
func TestCitesValueDispensa(t *testing.T) {
	yes := []string{
		"baseada no art. 24, inciso II da Lei Federal nº. 8.666",
		"fundamento legal o art. 24, inciso II c/c 23, II da Lei nº 8.666/93",
		"conforme artigo 75, inciso II da Lei de Licitações n.º 14.133",
		"nos termos do art. 75, II, da Lei 14.133",
		"Art. 75, caput, inciso II",
	}
	no := []string{
		"art. 24, inciso XIII, da Lei Federal nº 8.666/93",
		"nos termos do artigo 24, inciso XXII, da Lei 8.666",
		"art. 75, inciso III",
		"art. 75, inciso VIII",
		"Dispensa de Licitação",
	}
	for _, s := range yes {
		if !CitesValueDispensa(s) {
			t.Errorf("deveria citar dispensa por valor: %q", s)
		}
	}
	for _, s := range no {
		if CitesValueDispensa(s) {
			t.Errorf("não deveria citar dispensa por valor: %q", s)
		}
	}
}

func TestDispensaTextFlags(t *testing.T) {
	if !CitesLei14133("Lei Federal nº 14.133/2021") || CitesLei14133("Lei 8.666") {
		t.Error("CitesLei14133")
	}
	if !CitesEmergency("contratação emergencial") || !CitesEmergency("situação de EMERGÊNCIA") || !CitesEmergency("calamidade pública") || CitesEmergency("aquisição de vidros") {
		t.Error("CitesEmergency")
	}
	if !IsRepublication("Republicado por incorreção da PMSG.") || IsRepublication("publicado em 20/05") {
		t.Error("IsRepublication")
	}
}

func TestDispensaLimitCents(t *testing.T) {
	cases := []struct {
		day   time.Time
		cites bool
		want  int64
	}{
		{civilDate(2016, 5, 1), false, 800000},
		{civilDate(2018, 7, 18), false, 800000},
		{civilDate(2018, 7, 19), false, 1760000},
		{civilDate(2021, 3, 31), true, 1760000},
		{civilDate(2021, 4, 1), true, 5000000},
		{civilDate(2022, 6, 1), false, 1760000},
		{civilDate(2023, 6, 1), true, 5720833},
		{civilDate(2023, 12, 29), false, 1760000},
		{civilDate(2023, 12, 30), false, 5720833},
		{civilDate(2024, 3, 1), false, 5990602},
		{civilDate(2025, 3, 1), false, 6272559},
		{civilDate(2026, 3, 1), false, 6549211},
	}
	for _, c := range cases {
		if got := DispensaLimitCents(c.day, c.cites); got != c.want {
			t.Errorf("%s (14.133=%v): %d, esperava %d", c.day.Format("2006-01-02"), c.cites, got, c.want)
		}
	}
}

func TestFormatBRL(t *testing.T) {
	for cents, want := range map[int64]string{0: "R$ 0,00", 5: "R$ 0,05", 1186200: "R$ 11.862,00", 2891160: "R$ 28.911,60", 654921100: "R$ 6.549.211,00"} {
		if got := FormatBRL(cents); got != want {
			t.Errorf("FormatBRL(%d) = %q, esperava %q", cents, got, want)
		}
	}
}
```

- [ ] Implementação:

```go
package domain

import "regexp"

var (
	valueDispensaRe = regexp.MustCompile(`(?i)art(?:igo)?\.?\s*(?:24|75)\s*,?\s*(?:caput\s*,?\s*)?(?:inciso\s*|inc\.\s*)?II(?:[^IVX]|$)`)
	lei14133Re      = regexp.MustCompile(`14\.133`)
	emergencyRe     = regexp.MustCompile(`(?i)emerg[êe]nc|calamidade`)
	republicationRe = regexp.MustCompile(`(?i)republicad[oa]`)
)

func CitesValueDispensa(body string) bool { return valueDispensaRe.MatchString(body) }
func CitesLei14133(body string) bool      { return lei14133Re.MatchString(body) }
func CitesEmergency(body string) bool     { return emergencyRe.MatchString(body) }
func IsRepublication(body string) bool    { return republicationRe.MatchString(body) }
```

```go
package domain

import "time"

type dispensaLimit struct {
	from  time.Time
	cents int64
}

var (
	lei8666DispensaLimits = []dispensaLimit{
		{civilDate(2018, 7, 19), 1760000},
		{time.Time{}, 800000},
	}
	lei14133DispensaLimits = []dispensaLimit{
		{civilDate(2026, 1, 1), 6549211},
		{civilDate(2025, 1, 1), 6272559},
		{civilDate(2024, 1, 1), 5990602},
		{civilDate(2023, 1, 1), 5720833},
		{time.Time{}, 5000000},
	}
	lei14133Start = civilDate(2021, 4, 1)
	lei8666End    = civilDate(2023, 12, 30)
)

func DispensaLimitCents(published time.Time, citesLei14133 bool) int64 {
	table := lei8666DispensaLimits
	if !published.Before(lei8666End) || (citesLei14133 && !published.Before(lei14133Start)) {
		table = lei14133DispensaLimits
	}
	for _, l := range table {
		if !published.Before(l.from) {
			return l.cents
		}
	}
	return 0
}

func civilDate(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
```

```go
package domain

import (
	"fmt"
	"strconv"
)

func FormatBRL(cents int64) string {
	whole := strconv.FormatInt(cents/100, 10)
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "." + whole[i:]
	}
	return fmt.Sprintf("R$ %s,%02d", whole, cents%100)
}
```

- [ ] `go test ./internal/core/domain/` → PASS. Commit `feat(api): limites e regras de texto da dispensa por valor`.

### Task 2: regra de fracionamento (domínio)

**Files:** Create `services/api/internal/core/domain/split_dispensa.go` e teste.

**Consumes:** Task 1. `PublicBody`, `EntityLabel`.

**Produces:**

```go
type DispensaProcess struct{ Key, Label string }

type DispensaAct struct {
	ActID       string
	CNPJ        string
	Processes   []DispensaProcess
	Organ       string
	PublishedAt time.Time
	ValueCents  int64
	Body        string
}

type DispensaContract struct {
	Processes      []DispensaProcess
	FirstPublished time.Time
	ValueCents     int64
	LimitCents     int64
	Organs         []string
	ActIDs         []string
}

type SplitDispensa struct {
	CNPJ       string
	Year       int
	Contracts  []DispensaContract
	TotalCents int64
	LimitCents int64
}

func FindSplitDispensas(acts []DispensaAct) []SplitDispensa
```

- [ ] Testes (falham), com os textos reais resumidos:

```go
const (
	semmaRatificacao = "RATIFICO a situação de dispensa de licitação: Processo nº: 21.577/2019. Valor Global: R$ 11.862,00. O presente Termo tem por fundamento legal o art. 24, inciso II c/c 23, II da Lei nº 8.666/93."
	semadRatificacao = "RATIFICO a situação de Dispensa de Licitação, baseada no art. 24, inciso II da Lei Federal nº. 8.666 de 21 de junho de 1993, Processo nº 53.639/2018, no valor de R$ 17.049,60."
	semadRepublicada = semadRatificacao + " Republicado por incorreção da PMSG."
	semtranLei14133  = "RATIFICO a DISPENSA DE LICITAÇÃO, conforme artigo 75, inciso II da Lei de Licitações n.º 14.133 de 1º de abril de 2021. VALOR TOTAL: R$ 57.999,60"
)

func dispensaAct(id, cnpj, organ string, day time.Time, cents int64, body string, processes ...string) DispensaAct {
	a := DispensaAct{ActID: id, CNPJ: cnpj, Organ: organ, PublishedAt: day, ValueCents: cents, Body: body}
	for _, p := range processes {
		a.Processes = append(a.Processes, DispensaProcess{Key: nonDigitRe.ReplaceAllString(p, ""), Label: p})
	}
	return a
}

func TestFindSplitDispensasFlagsTheSameSupplierAboveTheYearlyLimit(t *testing.T) {
	acts := []DispensaAct{
		dispensaAct("semma", "53775862000152", "SEMMA", civilDate(2020, 2, 19), 1186200, semmaRatificacao, "21.577/2019"),
		dispensaAct("semad", "53775862000152", "SEMAD", civilDate(2020, 5, 22), 1704960, semadRatificacao, "53.639/2018"),
		dispensaAct("semad-rep", "53775862000152", "SEMAD", civilDate(2020, 5, 28), 1704960, semadRepublicada, "56.639/2018"),
	}

	got := FindSplitDispensas(acts)

	if len(got) != 1 {
		t.Fatalf("esperava 1 caso, veio %+v", got)
	}
	s := got[0]
	if s.CNPJ != "53775862000152" || s.Year != 2020 || len(s.Contracts) != 2 || s.TotalCents != 2891160 || s.LimitCents != 1760000 {
		t.Fatalf("caso inesperado: %+v", s)
	}
	if ids := s.Contracts[1].ActIDs; len(ids) != 2 || ids[0] != "semad" || ids[1] != "semad-rep" {
		t.Errorf("a republicação deveria entrar na contratação da SEMAD: %+v", s.Contracts[1])
	}
}

func TestFindSplitDispensasCountsOneContractPerProcess(t *testing.T) {
	cnpj := "18657198000146"
	acts := []DispensaAct{
		dispensaAct("ratificacao", cnpj, "SEMTRAN", civilDate(2024, 10, 3), 5799960, semtranLei14133, "18.635/2024"),
		dispensaAct("termo", cnpj, "SEMTRAN", civilDate(2024, 10, 4), 5799960, semtranLei14133, "18.635/2024"),
		dispensaAct("contrato", cnpj, "SEMTRAN", civilDate(2024, 10, 9), 5799960, "Valor Global: R$ 57.999,60", "18.635/2024"),
	}

	if got := FindSplitDispensas(acts); len(got) != 0 {
		t.Fatalf("o mesmo processo publicado três vezes não é fracionamento: %+v", got)
	}
}

func TestFindSplitDispensasJoinsProcessesCitedByTheSameAct(t *testing.T) {
	cnpj := "63832662000148"
	body := "com fundamento no art. 75, inciso II, da Lei Federal nº 14.133/2021. Valor Total: R$ 53.000,00"
	acts := []DispensaAct{
		dispensaAct("edital-colado", cnpj, "SEMHAB", civilDate(2026, 7, 30), 5300000, body, "7405/2026", "07537/2026"),
		dispensaAct("contrato", cnpj, "SMTC", civilDate(2026, 9, 3), 5300000, body, "7405/2026"),
		dispensaAct("outro", cnpj, "SMTC", civilDate(2026, 9, 10), 5300000, body, "07537/2026"),
	}

	if got := FindSplitDispensas(acts); len(got) != 0 {
		t.Fatalf("processos citados no mesmo ato são uma contratação: %+v", got)
	}
}

func TestFindSplitDispensasJoinsActWithoutProcessBySameValue(t *testing.T) {
	cnpj := "11222333000181"
	body := "art. 75, inciso II, da Lei 14.133. Valor: R$ 40.000,00"
	acts := []DispensaAct{
		dispensaAct("a", cnpj, "SEMED", civilDate(2025, 2, 1), 4000000, body, "1.111/2025"),
		dispensaAct("a-sem-processo", cnpj, "SEMED", civilDate(2025, 2, 3), 4000000, body),
	}

	if got := FindSplitDispensas(acts); len(got) != 0 {
		t.Fatalf("ato sem processo com o mesmo valor é a mesma contratação: %+v", got)
	}
}

func TestFindSplitDispensasIgnoresWhatIsNotAValueDispensa(t *testing.T) {
	cnpj := "11222333000181"
	value := "art. 75, inciso II, da Lei 14.133."
	cases := map[string][]DispensaAct{
		"emergência": {
			dispensaAct("a", cnpj, "SEMED", civilDate(2025, 2, 1), 4000000, value, "1.111/2025"),
			dispensaAct("b", cnpj, "SEMED", civilDate(2025, 3, 1), 4000000, value+" contratação emergencial", "2.222/2025"),
		},
		"outro inciso": {
			dispensaAct("a", cnpj, "SEMED", civilDate(2025, 2, 1), 4000000, value, "1.111/2025"),
			dispensaAct("b", cnpj, "SEMED", civilDate(2025, 3, 1), 4000000, "art. 75, inciso VIII", "2.222/2025"),
		},
		"acima do limite": {
			dispensaAct("a", cnpj, "SEMED", civilDate(2025, 2, 1), 4000000, value, "1.111/2025"),
			dispensaAct("b", cnpj, "SEMED", civilDate(2025, 3, 1), 7000000, value, "2.222/2025"),
		},
		"órgão público": {
			dispensaAct("a", "39260120000163", "SEMED", civilDate(2025, 2, 1), 4000000, value, "1.111/2025"),
			dispensaAct("b", "39260120000163", "SEMED", civilDate(2025, 3, 1), 4000000, value, "2.222/2025"),
		},
		"anos diferentes": {
			dispensaAct("a", cnpj, "SEMED", civilDate(2024, 12, 20), 4000000, value, "1.111/2024"),
			dispensaAct("b", cnpj, "SEMED", civilDate(2025, 1, 10), 4000000, value, "2.222/2025"),
		},
	}
	for name, acts := range cases {
		if got := FindSplitDispensas(acts); len(got) != 0 {
			t.Errorf("%s: não deveria acionar, veio %+v", name, got)
		}
	}
}
```

- [ ] Implementação:

```go
package domain

import (
	"sort"
	"time"
)

func FindSplitDispensas(acts []DispensaAct) []SplitDispensa {
	byCNPJ := map[string][]DispensaAct{}
	for _, a := range acts {
		if _, public := PublicBody(a.CNPJ); public || CitesEmergency(a.Body) || a.ValueCents <= 0 {
			continue
		}
		byCNPJ[a.CNPJ] = append(byCNPJ[a.CNPJ], a)
	}
	var out []SplitDispensa
	for cnpj, list := range byCNPJ {
		byYear := map[int][]DispensaContract{}
		for _, c := range valueDispensaContracts(list) {
			byYear[c.FirstPublished.Year()] = append(byYear[c.FirstPublished.Year()], c)
		}
		for year, contracts := range byYear {
			if s, ok := splitOf(cnpj, year, contracts); ok {
				out = append(out, s)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Year != out[j].Year {
			return out[i].Year < out[j].Year
		}
		return out[i].CNPJ < out[j].CNPJ
	})
	return out
}

func splitOf(cnpj string, year int, contracts []DispensaContract) (SplitDispensa, bool) {
	s := SplitDispensa{CNPJ: cnpj, Year: year, Contracts: contracts}
	for _, c := range contracts {
		s.TotalCents += c.ValueCents
		s.LimitCents = max(s.LimitCents, c.LimitCents)
	}
	sort.Slice(s.Contracts, func(i, j int) bool { return s.Contracts[i].FirstPublished.Before(s.Contracts[j].FirstPublished) })
	return s, len(contracts) >= 2 && s.TotalCents > s.LimitCents
}

func valueDispensaContracts(acts []DispensaAct) []DispensaContract {
	sort.Slice(acts, func(i, j int) bool {
		if !acts[i].PublishedAt.Equal(acts[j].PublishedAt) {
			return acts[i].PublishedAt.Before(acts[j].PublishedAt)
		}
		return acts[i].ActID < acts[j].ActID
	})
	groups := newUnionFind(len(acts))
	firstByProcess := map[string]int{}
	for i, a := range acts {
		for _, p := range a.Processes {
			if j, ok := firstByProcess[p.Key]; ok {
				groups.union(i, j)
			} else {
				firstByProcess[p.Key] = i
			}
		}
	}
	for i, a := range acts {
		if len(a.Processes) > 0 && !IsRepublication(a.Body) {
			continue
		}
		if j, ok := sameValueSameYear(acts, i, groups); ok {
			groups.union(i, j)
		}
	}
	var out []DispensaContract
	for _, members := range groups.sets() {
		if c, ok := contractOf(acts, members); ok {
			out = append(out, c)
		}
	}
	return out
}

func sameValueSameYear(acts []DispensaAct, i int, groups *unionFind) (int, bool) {
	a := acts[i]
	for j, b := range acts {
		if groups.find(j) != groups.find(i) && b.ValueCents == a.ValueCents && b.PublishedAt.Year() == a.PublishedAt.Year() {
			return j, true
		}
	}
	return 0, false
}

func contractOf(acts []DispensaAct, members []int) (DispensaContract, bool) {
	var c DispensaContract
	valueDispensa := false
	seenProcess, seenOrgan := map[string]bool{}, map[string]bool{}
	for _, i := range members {
		a := acts[i]
		if c.FirstPublished.IsZero() || a.PublishedAt.Before(c.FirstPublished) {
			c.FirstPublished = a.PublishedAt
		}
		c.ValueCents = max(c.ValueCents, a.ValueCents)
		c.LimitCents = max(c.LimitCents, DispensaLimitCents(a.PublishedAt, CitesLei14133(a.Body)))
		valueDispensa = valueDispensa || CitesValueDispensa(a.Body)
		c.ActIDs = append(c.ActIDs, a.ActID)
		for _, p := range a.Processes {
			if !seenProcess[p.Key] {
				seenProcess[p.Key] = true
				c.Processes = append(c.Processes, p)
			}
		}
		if a.Organ != "" && !seenOrgan[a.Organ] {
			seenOrgan[a.Organ] = true
			c.Organs = append(c.Organs, a.Organ)
		}
	}
	return c, valueDispensa && c.ValueCents < c.LimitCents
}

type unionFind struct{ parent []int }

func newUnionFind(n int) *unionFind {
	u := &unionFind{parent: make([]int, n)}
	for i := range u.parent {
		u.parent[i] = i
	}
	return u
}

func (u *unionFind) find(i int) int {
	for u.parent[i] != i {
		u.parent[i] = u.parent[u.parent[i]]
		i = u.parent[i]
	}
	return i
}

func (u *unionFind) union(i, j int) {
	ri, rj := u.find(i), u.find(j)
	if ri < rj {
		u.parent[rj] = ri
	} else if rj < ri {
		u.parent[ri] = rj
	}
}

func (u *unionFind) sets() [][]int {
	byRoot := map[int][]int{}
	var roots []int
	for i := range u.parent {
		r := u.find(i)
		if _, ok := byRoot[r]; !ok {
			roots = append(roots, r)
		}
		byRoot[r] = append(byRoot[r], i)
	}
	out := make([][]int, 0, len(roots))
	for _, r := range roots {
		out = append(out, byRoot[r])
	}
	return out
}
```

(`time` só é usado pelos tipos; os tipos ficam neste arquivo.)

- [ ] `go test ./internal/core/domain/` → PASS. Commit `feat(api): regra de fracionamento de dispensas por fornecedor e ano`.

### Task 3: regra de pico antes da eleição (domínio)

**Files:** Create `services/api/internal/core/domain/election_hiring.go` e teste.

**Produces:**

```go
var MunicipalElections = map[int]time.Time{2012: civilDate(2012, 10, 7), 2016: civilDate(2016, 10, 2), 2020: civilDate(2020, 11, 15), 2024: civilDate(2024, 10, 6)}

type MonthlyActCount struct {
	Type  ActType
	Year  int
	Month time.Month
	Count int
}

type HiringPeak struct {
	Type           ActType
	Year           int
	Month          time.Month
	Count          int
	BaselineMedian float64
	BaselineYears  int
	Election       time.Time
}

func FindElectionPeaks(counts []MonthlyActCount) []HiringPeak
```

- [ ] Testes (falham):

```go
func TestFindElectionPeaks(t *testing.T) {
	var counts []MonthlyActCount
	for _, y := range []int{2013, 2014, 2015, 2017, 2018} {
		counts = append(counts, MonthlyActCount{ActNomeacao, y, time.August, 100}, MonthlyActCount{ActNomeacao, y, time.November, 100})
	}
	counts = append(counts,
		MonthlyActCount{ActNomeacao, 2020, time.August, 150},
		MonthlyActCount{ActNomeacao, 2016, time.August, 149},
		MonthlyActCount{ActNomeacao, 2012, time.November, 400},
		MonthlyActCount{ActNomeacao, 2024, time.August, 1000},
		MonthlyActCount{ActNomeacao, 2024, time.November, 1000},
	)

	got := FindElectionPeaks(counts)

	if len(got) != 2 {
		t.Fatalf("esperava 2 picos (ago/2020 e ago/2024), veio %+v", got)
	}
	p := got[0]
	if p.Year != 2020 || p.Month != time.August || p.Count != 150 || p.BaselineMedian != 100 || p.BaselineYears != 5 || !p.Election.Equal(civilDate(2020, 11, 15)) {
		t.Errorf("pico inesperado: %+v", p)
	}
	if got[1].Year != 2024 || got[1].Month != time.August {
		t.Errorf("segundo pico inesperado: %+v", got[1])
	}
}

func TestFindElectionPeaksNeedsABaseline(t *testing.T) {
	counts := []MonthlyActCount{{ActExoneracao, 2020, time.August, 500}}

	if got := FindElectionPeaks(counts); len(got) != 0 {
		t.Fatalf("sem anos de comparação não há pico: %+v", got)
	}
}
```

(Novembro de 2012 é o mês seguinte à eleição de 2012, fora da janela; novembro de 2024, idem; 149 < 150.)

- [ ] Implementação:

```go
const (
	electionWindowMonths = 6
	electionPeakRatio    = 1.5
)

type monthKey struct {
	t ActType
	y int
	m time.Month
}

func FindElectionPeaks(counts []MonthlyActCount) []HiringPeak {
	byKey := map[monthKey]int{}
	types := map[ActType]bool{}
	for _, c := range counts {
		byKey[monthKey{c.Type, c.Year, c.Month}] = c.Count
		types[c.Type] = true
	}
	var out []HiringPeak
	for t := range types {
		for year, election := range MunicipalElections {
			first := civilDate(election.Year(), election.Month(), 1)
			for k := electionWindowMonths; k >= 1; k-- {
				day := first.AddDate(0, -k, 0)
				baseline := baselineCounts(byKey, t, day.Month())
				if len(baseline) == 0 {
					continue
				}
				median := medianOf(baseline)
				n := byKey[monthKey{t, day.Year(), day.Month()}]
				if n > 0 && float64(n) >= electionPeakRatio*median {
					out = append(out, HiringPeak{Type: t, Year: day.Year(), Month: day.Month(), Count: n,
						BaselineMedian: median, BaselineYears: len(baseline), Election: MunicipalElections[year]})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Year != out[j].Year {
			return out[i].Year < out[j].Year
		}
		if out[i].Month != out[j].Month {
			return out[i].Month < out[j].Month
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func baselineCounts(byKey map[monthKey]int, t ActType, m time.Month) []int {
	var out []int
	for k, n := range byKey {
		if _, election := MunicipalElections[k.y]; k.t == t && k.m == m && !election {
			out = append(out, n)
		}
	}
	return out
}

func medianOf(xs []int) float64 {
	s := append([]int(nil), xs...)
	sort.Ints(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return float64(s[mid])
	}
	return float64(s[mid-1]+s[mid]) / 2
}
```

- [ ] `go test ./internal/core/domain/` → PASS. Commit `feat(api): regra de pico de nomeações e exonerações antes da eleição`.

### Task 4: catálogo de padrões e textos dos casos (domínio)

**Files:** Create `services/api/internal/core/domain/pattern.go` e teste.

**Produces:**

```go
type PatternID string

const (
	PatternSplitDispensa  PatternID = "fracionamento_dispensa"
	PatternElectionHiring PatternID = "pico_pessoal_eleicao"
)

type Pattern struct {
	ID                   PatternID
	Title, Rule, Caveat  string
}

type Finding struct {
	Title  string
	Detail string
	ActIDs []string
	Search *ActFilter
}

type PatternReport struct {
	Pattern  Pattern
	Findings []Finding
}

func PatternCatalog() map[PatternID]Pattern
func SplitDispensaFinding(s SplitDispensa) Finding
func ElectionPeakFinding(p HiringPeak) Finding
```

- [ ] Testes (falham):

```go
func TestSplitDispensaFinding(t *testing.T) {
	s := SplitDispensa{CNPJ: "53775862000152", Year: 2020, TotalCents: 2891160, LimitCents: 1760000, Contracts: []DispensaContract{
		{Processes: []DispensaProcess{{"215772019", "21.577/2019"}}, FirstPublished: civilDate(2020, 2, 19), ValueCents: 1186200, Organs: []string{"SEMMA"}, ActIDs: []string{"a"}},
		{Processes: []DispensaProcess{{"536392018", "53.639/2018"}, {"566392018", "56.639/2018"}}, FirstPublished: civilDate(2020, 5, 22), ValueCents: 1704960, Organs: []string{"SEMAD"}, ActIDs: []string{"b", "c"}},
	}}

	f := SplitDispensaFinding(s)

	if f.Title != "CNPJ 53.775.862/0001-52 em 2020: 2 dispensas somam R$ 28.911,60, acima do limite de R$ 17.600,00" {
		t.Errorf("título: %q", f.Title)
	}
	if f.Detail != "Processo 21.577/2019 (SEMMA), 19/02/2020: R$ 11.862,00. Processos 53.639/2018 e 56.639/2018 (SEMAD), 22/05/2020: R$ 17.049,60." {
		t.Errorf("detalhe: %q", f.Detail)
	}
	if len(f.ActIDs) != 3 || f.Search != nil {
		t.Errorf("atos: %+v", f)
	}
}

func TestElectionPeakFinding(t *testing.T) {
	p := HiringPeak{Type: ActNomeacao, Year: 2020, Month: time.August, Count: 166, BaselineMedian: 107, BaselineYears: 13, Election: civilDate(2020, 11, 15)}

	f := ElectionPeakFinding(p)

	if f.Title != "Agosto de 2020: 166 atos de nomeação" ||
		f.Detail != "Mediana de agosto nos 13 anos sem eleição municipal: 107. Eleição em 15/11/2020." {
		t.Errorf("caso inesperado: %+v", f)
	}
	if f.Search == nil || f.Search.Type != ActNomeacao || f.Search.Source != SourceDiarioPrefeitura ||
		!f.Search.From.Equal(civilDate(2020, 8, 1)) || !f.Search.To.Equal(civilDate(2020, 8, 31)) {
		t.Errorf("busca inesperada: %+v", f.Search)
	}
}
```

(Mediana com meio, como 67,5, sai "67,5": `strconv.FormatFloat(m, 'f', -1, 64)` com ponto trocado por vírgula.)

- [ ] Implementação:

```go
var monthNames = [...]string{"", "janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}

var peakTypeLabel = map[ActType]string{ActNomeacao: "nomeação", ActExoneracao: "exoneração"}

func PatternCatalog() map[PatternID]Pattern {
	return map[PatternID]Pattern{
		PatternSplitDispensa: {
			ID:    PatternSplitDispensa,
			Title: "Dispensas do mesmo fornecedor que, somadas no ano, passam do limite",
			Rule: "Duas ou mais contratações do mesmo CNPJ no mesmo ano, cada uma por dispensa de licitação em razão do valor " +
				"(art. 24, II, da Lei 8.666 ou art. 75, II, da Lei 14.133), cada uma abaixo do limite da dispensa e com soma acima dele. " +
				"Uma contratação reúne os atos que citam o mesmo processo; republicações e atos sem número de processo com o mesmo valor " +
				"no mesmo ano contam como a mesma contratação. Limite de compras e serviços: R$ 8.000,00 até 18/07/2018, R$ 17.600,00 " +
				"depois, e pela Lei 14.133 R$ 50.000,00, atualizados todo ano (R$ 65.492,11 em 2026).",
			Caveat: "A lei soma o que cada unidade gestora gasta no ano com objetos de mesma natureza. O Diário não diz a natureza do " +
				"objeto de forma padronizada, e o mesmo fornecedor pode vender coisas diferentes para órgãos diferentes. Só entram atos " +
				"em que o CNPJ e o valor foram lidos do texto, então parte das dispensas fica de fora.",
		},
		PatternElectionHiring: {
			ID:    PatternElectionHiring,
			Title: "Picos de nomeação e de exoneração antes da eleição municipal",
			Rule: "Nos seis meses anteriores ao mês de cada eleição municipal (2012, 2016, 2020 e 2024), os meses em que o Diário da " +
				"Prefeitura publicou pelo menos 1,5 vez a mediana de atos de nomeação (ou de exoneração) do mesmo mês nos anos sem " +
				"eleição municipal.",
			Caveat: "Conta atos, não pessoas: uma portaria pode nomear ou exonerar várias. Um pico pode ter explicação comum, como troca " +
				"de secretário ou posse de concursados; o padrão só aponta o mês para verificar.",
		},
	}
}

func SplitDispensaFinding(s SplitDispensa) Finding {
	f := Finding{Title: fmt.Sprintf("CNPJ %s em %d: %d dispensas somam %s, acima do limite de %s",
		FormatCNPJ(s.CNPJ), s.Year, len(s.Contracts), FormatBRL(s.TotalCents), FormatBRL(s.LimitCents))}
	parts := make([]string, 0, len(s.Contracts))
	for _, c := range s.Contracts {
		parts = append(parts, contractSummary(c))
		f.ActIDs = append(f.ActIDs, c.ActIDs...)
	}
	f.Detail = strings.Join(parts, " ")
	return f
}

func contractSummary(c DispensaContract) string {
	labels := make([]string, 0, len(c.Processes))
	for _, p := range c.Processes {
		labels = append(labels, p.Label)
	}
	var who string
	switch len(labels) {
	case 0:
		who = "Sem número de processo"
	case 1:
		who = "Processo " + labels[0]
	default:
		who = "Processos " + strings.Join(labels[:len(labels)-1], ", ") + " e " + labels[len(labels)-1]
	}
	if len(c.Organs) > 0 {
		who += " (" + strings.Join(c.Organs, ", ") + ")"
	}
	return fmt.Sprintf("%s, %s: %s.", who, c.FirstPublished.Format("02/01/2006"), FormatBRL(c.ValueCents))
}

func ElectionPeakFinding(p HiringPeak) Finding {
	first := civilDate(p.Year, p.Month, 1)
	month := monthNames[p.Month]
	return Finding{
		Title: fmt.Sprintf("%s de %d: %d atos de %s", strings.ToUpper(month[:1])+month[1:], p.Year, p.Count, peakTypeLabel[p.Type]),
		Detail: fmt.Sprintf("Mediana de %s nos %d anos sem eleição municipal: %s. Eleição em %s.",
			month, p.BaselineYears, strings.Replace(strconv.FormatFloat(p.BaselineMedian, 'f', -1, 64), ".", ",", 1), p.Election.Format("02/01/2006")),
		Search: &ActFilter{Type: p.Type, Source: SourceDiarioPrefeitura, From: first, To: first.AddDate(0, 1, -1)},
	}
}
```

(`month[:1]` é seguro: todos os nomes começam com letra ASCII.)

- [ ] `go test ./internal/core/domain/` → PASS. Commit `feat(api): catálogo de padrões e texto de cada caso`.

### Task 5: Postgres, caso de uso e `/v1/patterns`

**Files:**
- Create `services/api/internal/adapters/postgres/patterns.go`, `services/api/internal/core/usecase/list_patterns.go` (+ teste), `services/api/internal/presentation/http/patterns.go`, `services/api/internal/integration/patterns_test.go`.
- Modify `services/api/internal/core/ports/ports.go`, `services/api/internal/presentation/http/router.go`, `services/api/cmd/api/main.go`, `services/api/internal/integration/investigator_test.go`.

**Produces:**
- `ports.PatternSource{ DispensaActs(ctx) ([]domain.DispensaAct, error); MonthlyActCounts(ctx, types []domain.ActType, source string) ([]domain.MonthlyActCount, error); HitsByIDs(ctx, ids []string) ([]domain.ActHit, error) }`
- `usecase.ListPatterns.Execute(ctx) ([]domain.PatternReport, map[string]domain.ActHit, error)`
- `API.Patterns *usecase.ListPatterns`; `GET /v1/patterns`.

- [ ] Teste do caso de uso (falha), com fonte falsa:

```go
type fakePatternSource struct {
	dispensas []domain.DispensaAct
	counts    []domain.MonthlyActCount
	askedIDs  []string
}

func (f *fakePatternSource) DispensaActs(context.Context) ([]domain.DispensaAct, error) { return f.dispensas, nil }
func (f *fakePatternSource) MonthlyActCounts(context.Context, []domain.ActType, string) ([]domain.MonthlyActCount, error) {
	return f.counts, nil
}
func (f *fakePatternSource) HitsByIDs(_ context.Context, ids []string) ([]domain.ActHit, error) {
	f.askedIDs = ids
	var out []domain.ActHit
	for _, id := range ids {
		out = append(out, domain.ActHit{Act: domain.Act{ID: id}})
	}
	return out, nil
}

func TestListPatternsReturnsEveryPatternWithItsFindingsAndActs(t *testing.T) {
	body := "art. 75, inciso II, da Lei 14.133"
	day := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	src := &fakePatternSource{dispensas: []domain.DispensaAct{
		{ActID: "a", CNPJ: "11222333000181", PublishedAt: day, ValueCents: 4000000, Body: body, Processes: []domain.DispensaProcess{{Key: "1", Label: "1"}}},
		{ActID: "b", CNPJ: "11222333000181", PublishedAt: day.AddDate(0, 1, 0), ValueCents: 4000000, Body: body, Processes: []domain.DispensaProcess{{Key: "2", Label: "2"}}},
	}}

	reports, acts, err := NewListPatterns(src).Execute(context.Background())

	if err != nil || len(reports) != 2 {
		t.Fatalf("veio %+v %v", reports, err)
	}
	if reports[0].Pattern.ID != domain.PatternSplitDispensa || len(reports[0].Findings) != 1 || reports[1].Pattern.ID != domain.PatternElectionHiring {
		t.Fatalf("relatórios inesperados: %+v", reports)
	}
	if len(acts) != 2 || acts["a"].ID != "a" || len(src.askedIDs) != 2 {
		t.Fatalf("atos inesperados: %+v %v", acts, src.askedIDs)
	}
}
```

- [ ] Caso de uso:

```go
package usecase

type ListPatterns struct{ src ports.PatternSource }

func NewListPatterns(src ports.PatternSource) *ListPatterns { return &ListPatterns{src: src} }

func (uc *ListPatterns) Execute(ctx context.Context) ([]domain.PatternReport, map[string]domain.ActHit, error) {
	catalog := domain.PatternCatalog()
	dispensas, err := uc.src.DispensaActs(ctx)
	if err != nil {
		return nil, nil, err
	}
	counts, err := uc.src.MonthlyActCounts(ctx, []domain.ActType{domain.ActNomeacao, domain.ActExoneracao}, domain.SourceDiarioPrefeitura)
	if err != nil {
		return nil, nil, err
	}
	split := domain.PatternReport{Pattern: catalog[domain.PatternSplitDispensa], Findings: []domain.Finding{}}
	var ids []string
	for _, s := range domain.FindSplitDispensas(dispensas) {
		f := domain.SplitDispensaFinding(s)
		split.Findings = append(split.Findings, f)
		ids = append(ids, f.ActIDs...)
	}
	peaks := domain.PatternReport{Pattern: catalog[domain.PatternElectionHiring], Findings: []domain.Finding{}}
	for _, p := range domain.FindElectionPeaks(counts) {
		peaks.Findings = append(peaks.Findings, domain.ElectionPeakFinding(p))
	}
	acts := map[string]domain.ActHit{}
	if len(ids) > 0 {
		hits, err := uc.src.HitsByIDs(ctx, ids)
		if err != nil {
			return nil, nil, err
		}
		for _, h := range hits {
			acts[h.ID] = h
		}
	}
	return []domain.PatternReport{split, peaks}, acts, nil
}
```

- [ ] Postgres (`patterns.go`):

```go
type PatternRepo struct{ db *sql.DB }

func NewPatternRepo(db *sql.DB) *PatternRepo { return &PatternRepo{db: db} }

func (r *PatternRepo) DispensaActs(ctx context.Context) ([]domain.DispensaAct, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, e.key, a.organ, g.published_at, a.main_value_cents, a.body,
		       coalesce((SELECT array_agg(p.key || '|' || pl.evidence ORDER BY p.key)
		                 FROM entity_links pl JOIN entities p ON p.id = pl.entity_id AND p.kind = 'processo'
		                 WHERE pl.record_kind = $1 AND pl.record_id = a.id::text), '{}')
		FROM acts a
		JOIN gazettes g ON g.id = a.gazette_id
		JOIN entity_links l ON l.record_kind = $1 AND l.record_id = a.id::text
		JOIN entities e ON e.id = l.entity_id AND e.kind = 'cnpj'
		WHERE (a.type = 'dispensa' OR a.modality = 'dispensa') AND a.type <> 'aditivo'
		  AND a.main_value_cents > 0 AND g.source = $2`, domain.RecordAct, domain.SourceDiarioPrefeitura)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.DispensaAct
	for rows.Next() {
		var a domain.DispensaAct
		var processes []string
		if err := rows.Scan(&a.ActID, &a.CNPJ, &a.Organ, &a.PublishedAt, &a.ValueCents, &a.Body, pq.Array(&processes)); err != nil {
			return nil, err
		}
		for _, p := range processes {
			key, evidence, _ := strings.Cut(p, "|")
			a.Processes = append(a.Processes, domain.DispensaProcess{Key: key, Label: domain.EntityLabel(domain.EntityProcesso, evidence)})
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PatternRepo) MonthlyActCounts(ctx context.Context, types []domain.ActType, source string) ([]domain.MonthlyActCount, error) {
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = string(t)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.type, extract(year FROM g.published_at)::int, extract(month FROM g.published_at)::int, count(*)
		FROM acts a JOIN gazettes g ON g.id = a.gazette_id
		WHERE a.type = ANY($1) AND g.source = $2 AND g.published_at < date_trunc('month', now())
		GROUP BY 1, 2, 3`, pq.Array(names), source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MonthlyActCount
	for rows.Next() {
		var c domain.MonthlyActCount
		var typ string
		var month int
		if err := rows.Scan(&typ, &c.Year, &month, &c.Count); err != nil {
			return nil, err
		}
		c.Type, c.Month = domain.ActType(typ), time.Month(month)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PatternRepo) HitsByIDs(ctx context.Context, ids []string) ([]domain.ActHit, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, a.organ, coalesce(a.page_start, 0), coalesce(a.page_end, 0),
		       coalesce(a.modality, ''), coalesce(a.main_value_cents, 0),
		       g.edition_number, g.published_at, g.is_extra, g.source_url, g.checksum, g.source,
		       left(a.body, 280), `+cnpjsSubquery+`, `+valuesSubquery+`, `+mentionsSubquery+`
		FROM acts a JOIN gazettes g ON g.id = a.gazette_id
		WHERE a.id = ANY($1::uuid[])
		ORDER BY g.published_at, a.position`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []domain.ActHit
	for rows.Next() {
		var h domain.ActHit
		var typ string
		var mentions []string
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Position, &h.Organ, &h.PageStart, &h.PageEnd,
			&h.Modality, &h.MainValueCents, &h.EditionNumber, &h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Checksum, &h.Source,
			&h.Snippet, pq.Array(&h.CNPJs), pq.Array(&h.ValuesCents), pq.Array(&mentions)); err != nil {
			return nil, err
		}
		h.Type = domain.ActType(typ)
		h.Mentions = parseMentions(mentions)
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hits, markBodyFacts(ctx, r.db, hits)
}
```

- [ ] HTTP (`patterns.go`):

```go
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
		dto.Findings = append(dto.Findings, fd)
	}
	return dto
}
```

`router.go`: campo `Patterns *usecase.ListPatterns` e `mux.HandleFunc("GET /v1/patterns", a.listPatterns)`. `cmd/api/main.go`: `Patterns: usecase.NewListPatterns(postgres.NewPatternRepo(db))`.

- [ ] Integração (`patterns_test.go`): indexa três edições com o texto real resumido (19/02/2020 SEMMA, 22/05/2020 SEMAD, 28/05/2020 SEMAD republicada; 03/10/2024, 04/10/2024 e 09/10/2024 SEMTRAN) por um auxiliar `indexAt(t, db, day, text)` e `GET /v1/patterns`: o padrão `fracionamento_dispensa` tem 1 caso, com título começando por `CNPJ 53.775.862/0001-52 em 2020` e 3 atos; nenhum caso cita `18.657.198/0001-46`; `pico_pessoal_eleicao` vem com `findings: []`. `newServerFor` ganha `Patterns: usecase.NewListPatterns(postgres.NewPatternRepo(db))`.

- [ ] `make lint test` e `make test-integration` → PASS. Commit `feat(api): padrões para verificar em /v1/patterns`.

### Task 6: página `/padroes`

**Files:** Create `apps/web/src/PatternsPage.tsx`, `apps/web/src/patterns.ts`, `apps/web/src/patterns.test.ts`. Modify `apps/web/src/api.ts`, `apps/web/src/App.tsx`, `apps/web/src/SearchPage.tsx`, `apps/web/src/styles.css` (só se faltar estilo).

**Produces:** `listPatterns()`, tipos `Pattern`, `Finding`, `PatternSearch`; `patternSearchHref(s: PatternSearch): string`.

- [ ] Teste (falha):

```ts
import { describe, expect, it } from "vitest";
import { patternSearchHref } from "./patterns";

describe("patternSearchHref", () => {
  it("abre a busca do site com tipo, período e diário", () => {
    expect(patternSearchHref({ type: "nomeacao", from: "2020-08-01", to: "2020-08-31", source: "diario_prefeitura" }))
      .toBe("/?tipo=nomeacao&de=2020-08-01&ate=2020-08-31&fonte=diario_prefeitura");
  });
});
```

- [ ] `patterns.ts`:

```ts
import { PatternSearch } from "./api";

export function patternSearchHref(s: PatternSearch) {
  return `/?${new URLSearchParams({ tipo: s.type, de: s.from, ate: s.to, fonte: s.source })}`;
}
```

  (Conferir em `searchState.ts` que `tipo`, `de`, `ate` e `fonte` são os nomes lidos da URL.)

- [ ] `api.ts`: tipos `PatternSearch { type: ActType; from: string; to: string; source: Source }`, `Finding { title; detail; acts: ActHit[]; search: PatternSearch | null }`, `Pattern { id; title; rule; caveat; findings: Finding[] }` e `listPatterns()` → `request<{ items: Pattern[] }>("/v1/patterns")`.

- [ ] `PatternsPage.tsx`: `crumb` de volta para a busca, `masthead` com "Padrões para verificar", aviso "Padrões para verificar, não irregularidades: cada caso só diz que os atos seguem a regra abaixo. Confira sempre a edição original." e, por padrão, `<section>` com `h2` (título), `p` com a regra, `p.fineprint` com a ressalva, contagem de casos, e cada caso em `article` com `h3` (título), `p` (detalhe), `ol.timeline` de `Result` e, com `search`, link "Ver os atos na busca". Sem casos: "Nenhum caso na base hoje."
- [ ] `App.tsx`: `if (path === "/padroes") return <PatternsPage />;`. `SearchPage.tsx`: link `<a href="/padroes">Padrões para verificar</a>` junto de "Dados abertos" e "MCP".
- [ ] `npm run typecheck` e `npm test` → PASS; conferir a página com Playwright na base local. Commit `feat(web): página de padrões para verificar`.

### Task 7: documentação

- [ ] README: linha `GET /v1/patterns` na tabela da API.
- [ ] `docs/parser-findings.md`: "AUTORIZAÇÃO DA DESPESA E ADJUDICAÇÃO" não abre ato (exemplo de 30/07/2026, SMTC colada no edital 023/2026 da SEMHAB).
- [ ] Roadmap, Entrega 2: "Padrões para verificar" com as duas regras entregues, os números da base local e as duas que ficaram (aditivos, emergencial) com o motivo.
- [ ] Commit `docs(roadmap): padrões para verificar`.
