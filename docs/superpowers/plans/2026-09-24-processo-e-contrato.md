# Páginas de processo e de contrato: plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** páginas `/processo/{slug}` e `/contrato/{slug}` com a linha do tempo por fase e por órgão, links para elas na busca e a mesma informação na `entidade` do MCP.

**Architecture:** a fase é uma função pura de domínio aplicada pelo caso de uso `GetEntity`. O `LinkRepo` passa a devolver rótulo, órgãos, contagem por tipo e título e entidades citadas junto. A busca traz os processos e contratos de cada ato. A API ganha a rota genérica, o site ganha `EntityPage` e o MCP ganha os campos novos.

**Tech Stack:** Go 1.25, PostgreSQL 16, React + Vite + Vitest, go-sdk de MCP.

**Spec:** `docs/superpowers/specs/2026-09-24-processo-e-contrato-design.md`

## Global Constraints

- Nenhum comentário novo no código (AGENTS.md). Migrations aplicadas não mudam; este plano não cria migration.
- A chave de processo e contrato (ADR 0004), a página do CNPJ e a rota `/v1/entities/cnpj/{cnpj}` não mudam de comportamento.
- Limite de atos no relatório: 300 para processo e contrato, 100 para CNPJ.
- `Related`: até 20 por tipo relacionado, pelos que têm mais atos.
- Linguagem da página: "podem ser processos diferentes", nunca afirmar que são o mesmo processo.
- Conventional commits em português, sem link de sessão de agente.
- Antes de concluir: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.

---

### Task 1: fase do ato no domínio

**Files:**
- Create: `services/api/internal/core/domain/phase.go`, `services/api/internal/core/domain/phase_test.go`

**Interfaces:**
- Produces: `type Phase string`; constantes `PhaseLicitacao` (`licitacao`), `PhaseHomologacao` (`homologacao`), `PhaseAtaRegistroPrecos` (`ata_registro_precos`), `PhaseDispensa` (`dispensa`), `PhaseContrato` (`contrato`), `PhaseFiscal` (`fiscal`), `PhaseAditivo` (`aditivo`), `PhaseAjusteContas` (`ajuste_contas`), `PhaseRescisao` (`rescisao`), `PhaseOutro` (`outro`); `var Phases []Phase` nessa ordem; `func PhaseOf(t ActType, title string) Phase`.

- [ ] **Step 1: teste com títulos reais**

```go
package domain

import "testing"

func TestPhaseOf(t *testing.T) {
	cases := []struct {
		typ   ActType
		title string
		want  Phase
	}{
		{ActContrato, "EXTRATO DE DISTRATO DE CONTRATO", PhaseRescisao},
		{ActOutro, "TERMO DE RESCISÃO UNILATERAL", PhaseRescisao},
		{ActAditivo, "EXTRATO DO SEGUNDO TERMO ADITIVO AO CONTRATO", PhaseAditivo},
		{ActContrato, "EXTRATO DE TERMO ADITVO", PhaseAditivo},
		{ActAditivo, "EXTRATO DO PRIMEIRO TERMO DE APOSTILAMENTO (RERATIFICAÇÃO) DO CONTRATO 009/", PhaseAditivo},
		{ActContrato, "EXTRATO DE AJUSTE DE CONTAS E RECONHECIMENTO DE DÍVIDA", PhaseAjusteContas},
		{ActContrato, "EXTRATO DE NOMEAÇÃO DE FISCAIS", PhaseFiscal},
		{ActOutro, "SUBSTITUIÇÃO DE FISCAL DO CONTRATO Nº 17/2021", PhaseFiscal},
		{ActLicitacao, "HOMOLOGAÇÃO/ADJUDICAÇÃO - CONVITE Nº 001/2011", PhaseHomologacao},
		{ActLicitacao, "EXTRATO DA ATA DE REGISTRO DE PREÇOS Nº 12/2023", PhaseAtaRegistroPrecos},
		{ActContrato, "EXTRATO DE ATA DE REGISTRO DE PREÇO", PhaseAtaRegistroPrecos},
		{ActDispensa, "EXTRATO DE RATIFICAÇÃO", PhaseDispensa},
		{ActOutro, "TERMO DE RATIFICAÇÃO", PhaseDispensa},
		{ActLicitacao, "AVISO DE DISPENSA ELETRÔNICA Nº 90003/2025", PhaseDispensa},
		{ActContrato, "EXTRATO DE CONTRATO", PhaseContrato},
		{ActLicitacao, "AVISO DE LICITAÇÃO", PhaseLicitacao},
		{ActEdital, "EDITAL DE PREGÃO ELETRÔNICO Nº 5/2024", PhaseLicitacao},
		{ActAta, "ATA DA SESSÃO PÚBLICA", PhaseLicitacao},
		{ActDespacho, "DESPACHO DO SECRETÁRIO", PhaseOutro},
		{ActPortaria, "PORTARIA Nº 14/FMS/2022.", PhaseOutro},
		{ActOutro, "AUTO DE INFRAÇÃO Nº 12", PhaseOutro},
	}
	for _, c := range cases {
		if got := PhaseOf(c.typ, c.title); got != c.want {
			t.Errorf("%s %q: esperava %s, veio %s", c.typ, c.title, c.want, got)
		}
	}
}

func TestPhasesListsEveryPhaseOnce(t *testing.T) {
	seen := map[Phase]bool{}
	for _, p := range Phases {
		if seen[p] {
			t.Fatalf("%s repetida", p)
		}
		seen[p] = true
	}
	if len(Phases) != 10 || Phases[0] != PhaseLicitacao || Phases[len(Phases)-1] != PhaseOutro {
		t.Fatalf("ordem inesperada: %v", Phases)
	}
}
```

- [ ] **Step 2: rodar e ver falhar** — `cd services/api && go test ./internal/core/domain -run 'TestPhase' -count=1` → não compila (`PhaseOf` indefinida).

- [ ] **Step 3: implementar**

```go
package domain

import "regexp"

type Phase string

const (
	PhaseLicitacao         Phase = "licitacao"
	PhaseHomologacao       Phase = "homologacao"
	PhaseAtaRegistroPrecos Phase = "ata_registro_precos"
	PhaseDispensa          Phase = "dispensa"
	PhaseContrato          Phase = "contrato"
	PhaseFiscal            Phase = "fiscal"
	PhaseAditivo           Phase = "aditivo"
	PhaseAjusteContas      Phase = "ajuste_contas"
	PhaseRescisao          Phase = "rescisao"
	PhaseOutro             Phase = "outro"
)

var Phases = []Phase{
	PhaseLicitacao, PhaseHomologacao, PhaseAtaRegistroPrecos, PhaseDispensa, PhaseContrato,
	PhaseFiscal, PhaseAditivo, PhaseAjusteContas, PhaseRescisao, PhaseOutro,
}

var phaseRules = []struct {
	phase Phase
	types []ActType
	title *regexp.Regexp
}{
	{PhaseRescisao, nil, regexp.MustCompile(`(?i)rescis|distrato`)},
	{PhaseAditivo, []ActType{ActAditivo}, regexp.MustCompile(`(?i)aditi?vo|apostil`)},
	{PhaseAjusteContas, nil, regexp.MustCompile(`(?i)ajuste\s+de\s+contas|reconhecimento\s+de\s+d[íi]vida`)},
	{PhaseFiscal, nil, regexp.MustCompile(`(?i)\bfisca(?:l|is)\b`)},
	{PhaseHomologacao, nil, regexp.MustCompile(`(?i)homolog|adjudic`)},
	{PhaseAtaRegistroPrecos, nil, regexp.MustCompile(`(?i)registro\s+de\s+pre[çc]o`)},
	{PhaseDispensa, []ActType{ActDispensa}, regexp.MustCompile(`(?i)dispensa|inexigibilidade|ratific`)},
	{PhaseContrato, []ActType{ActContrato}, nil},
	{PhaseLicitacao, []ActType{ActLicitacao, ActEdital, ActAta}, nil},
}

func PhaseOf(t ActType, title string) Phase {
	for _, r := range phaseRules {
		if r.title != nil && r.title.MatchString(title) {
			return r.phase
		}
		for _, rt := range r.types {
			if rt == t {
				return r.phase
			}
		}
	}
	return PhaseOutro
}
```

Atenção à ordem: a regra do tipo só vale depois de o título não ter casado com uma regra anterior. Com o laço acima, `ActAditivo` casa em `PhaseAditivo` antes de chegar em `PhaseAjusteContas`, que é o desejado (aditivo de ajuste de contas é aditivo). `EXTRATO DE ATA DE REGISTRO DE PREÇO` com tipo `contrato` chega em `PhaseAtaRegistroPrecos` antes de `PhaseContrato`. `PORTARIA` que cita fiscal só no corpo não casa, porque só o título é olhado.

- [ ] **Step 4: rodar e ver passar** — mesmo comando → PASS.
- [ ] **Step 5: commit** — `feat(api): fase do ato no processo de contratação`.

### Task 2: rótulo, entrada pela URL e avisos da entidade

**Files:**
- Modify: `services/api/internal/core/domain/entity_key.go` (`ParseEntityInput`), `services/api/internal/core/domain/entity.go` (tipos novos)
- Create: `services/api/internal/core/domain/entity_label.go`, `entity_label_test.go`, `entity_warnings.go`, `entity_warnings_test.go`
- Modify: `services/api/internal/core/domain/entity_key_test.go` (casos com `-`)
- Modify: `services/api/internal/presentation/mcp/tools.go` (`certaintyWarning` passa a usar `domain.EntityWarnings`)

**Interfaces:**
- Produces:
  - `func EntityLabel(kind EntityKind, value string) string`
  - `func EntitySlug(label string) string` (troca `/` por `-`)
  - `type OrganCount struct { Organ string; Acts int }`
  - `type RelatedEntity struct { Kind EntityKind; Key, Label string; Acts int }`
  - `type EntityMention struct { Kind EntityKind; Key, Label string }`
  - `type TypeTitleCount struct { Type ActType; Title string; Acts int }`
  - campos novos em `EntityReport`: `Label string`, `Organs []OrganCount`, `CountByPhase map[Phase]int`, `Related []RelatedEntity`, `TypeTitleCounts []TypeTitleCount`
  - campos novos em `ActHit`: `Phase Phase`, `Mentions []EntityMention`
  - `func EntityWarnings(r EntityReport) []string`

- [ ] **Step 1: testes**

`entity_label_test.go`:

```go
func TestEntityLabel(t *testing.T) {
	cases := []struct {
		kind  EntityKind
		value string
		want  string
	}{
		{EntityProcesso, "PROCESSO ADMINISTRATIVO N.º 7148/2022", "7148/2022"},
		{EntityProcesso, "Processo n.º 14.672/2021", "14.672/2021"},
		{EntityProcesso, "Processo\n\n03.06524/2022-4", "03.06524/2022-4"},
		{EntityProcesso, "processo SEI Nº\n25.00383/2026-5", "25.00383/2026-5"},
		{EntityContrato, "Contrato PMSG Nº.\n001/2016", "001/2016"},
		{EntityContrato, "CONTRATO Nº 30/FMS/2011", "30/FMS/2011"},
		{EntityContrato, "Contrato nº 007/2024/SEMAD.", "007/2024/SEMAD"},
		{EntityCNPJ, "12345678000190", "12.345.678/0001-90"},
	}
	for _, c := range cases {
		if got := EntityLabel(c.kind, c.value); got != c.want {
			t.Errorf("%s %q: esperava %q, veio %q", c.kind, c.value, c.want, got)
		}
	}
}

func TestEntitySlugRoundTripsThroughParse(t *testing.T) {
	for kind, label := range map[EntityKind]string{EntityProcesso: "14.672/2021", EntityContrato: "30/FMS/2011"} {
		want, _ := ParseEntityInput(kind, label)
		got, err := ParseEntityInput(kind, EntitySlug(label))
		if err != nil || got != want {
			t.Errorf("%s %q: slug %q deu %q (%v), esperava %q", kind, label, EntitySlug(label), got, err, want)
		}
	}
}
```

Em `entity_key_test.go`, acrescentar `{EntityContrato, "30-fms-2011", "30/FMS/2011"}` e `{EntityProcesso, "387-2022", "3872022"}` aos casos válidos de `ParseEntityInput`.

`entity_warnings_test.go`:

```go
func TestEntityWarnings(t *testing.T) {
	many := EntityReport{Kind: EntityProcesso, Certainty: CertaintyStrong, Sources: 1,
		Organs: []OrganCount{{"SEMAD", 3}, {"SEMTRAN", 2}, {"", 1}}}
	if w := EntityWarnings(many); len(w) != 1 || !strings.Contains(w[0], "aparece em 2 órgãos") {
		t.Fatalf("vários órgãos: %v", w)
	}
	weak := EntityReport{Kind: EntityContrato, Certainty: CertaintyWeak, Sources: 1, Organs: []OrganCount{{"FMS", 1}}}
	if w := EntityWarnings(weak); len(w) != 1 || !strings.Contains(w[0], "sem a sigla do órgão") {
		t.Fatalf("certeza fraca: %v", w)
	}
	both := EntityReport{Kind: EntityProcesso, Certainty: CertaintyWeak, Sources: 2}
	if w := EntityWarnings(both); len(w) != 1 || !strings.Contains(w[0], "no Diário da Prefeitura e no da Câmara") {
		t.Fatalf("dois Diários: %v", w)
	}
	if w := EntityWarnings(EntityReport{Kind: EntityCNPJ, Certainty: CertaintyExact, Organs: []OrganCount{{"A", 1}, {"B", 1}}}); len(w) != 0 {
		t.Fatalf("CNPJ em vários órgãos é normal: %v", w)
	}
}
```

O órgão vazio (`""`, atos sem órgão) não conta como órgão.

- [ ] **Step 2: rodar e ver falhar** — `go test ./internal/core/domain -count=1`.

- [ ] **Step 3: implementar**

`entity_label.go`:

```go
package domain

import (
	"regexp"
	"strings"
)

var labelNumberRe = regexp.MustCompile(`\d[\d./A-Za-z-]*[\dA-Za-z]|\d`)

func EntityLabel(kind EntityKind, value string) string {
	if kind == EntityCNPJ {
		return FormatCNPJ(value)
	}
	matches := labelNumberRe.FindAllString(strings.Join(strings.Fields(value), " "), -1)
	if len(matches) == 0 {
		return strings.TrimSpace(value)
	}
	return strings.ToUpper(matches[len(matches)-1])
}

func EntitySlug(label string) string { return strings.ReplaceAll(label, "/", "-") }
```

Se `FormatCNPJ` não existir no domínio, criar em `entity_label.go` com o formato `00.000.000/0000-00` (confira antes com `grep -rn "func FormatCNPJ\|func formatCNPJ" services/api/internal`; se houver uma função não exportada em `presentation/mcp`, mover para o domínio e usá-la nos dois lugares).

`ParseEntityInput`, ramo contrato: `n := strings.ToUpper(strings.ReplaceAll(strings.Join(strings.Fields(s), ""), "-", "/"))`.

`entity_warnings.go`, com os textos que hoje estão em `certaintyWarning` do MCP:

```go
package domain

import "fmt"

func EntityWarnings(r EntityReport) []string {
	var out []string
	switch {
	case r.Kind != EntityCNPJ && r.Sources > 1:
		out = append(out, "Este número aparece no Diário da Prefeitura e no da Câmara, que numeram processos e contratos cada um à sua maneira: "+
			"podem ser registros diferentes. Use diario para ver só um dos dois e confira o campo diario de cada ato.")
	case r.Certainty == CertaintyWeak:
		out = append(out, "Número de contrato sem a sigla do órgão: estes atos podem ser de contratos diferentes, de órgãos diferentes, com o mesmo número e ano. Confira o órgão em cada ato.")
	}
	if n := namedOrgans(r.Organs); r.Kind != EntityCNPJ && n > 1 && r.Certainty != CertaintyWeak {
		out = append(out, fmt.Sprintf("Este número aparece em %d órgãos. Podem ser processos diferentes com o mesmo número, "+
			"ou uma ata de registro de preços usada por vários órgãos. Confira o órgão em cada ato.", n))
	}
	return out
}

func namedOrgans(os []OrganCount) int {
	n := 0
	for _, o := range os {
		if o.Organ != "" {
			n++
		}
	}
	return n
}
```

O aviso de vários órgãos não se soma ao de certeza fraca, que já diz a mesma coisa. No MCP, `certaintyWarning(report)` vira `strings.Join(domain.EntityWarnings(report), " ")`; a função `certaintyWarning` sai.

- [ ] **Step 4: rodar** — `go test ./internal/core/domain ./internal/presentation/mcp -count=1` → PASS.
- [ ] **Step 5: commit** — `feat(api): rótulo, slug e avisos de entidade no domínio`.

### Task 3: menções de processo e contrato em cada ato

**Files:**
- Modify: `services/api/internal/adapters/postgres/search.go` (subconsulta `mentionsSubquery` e conversão), `acts.go` (`Search`), `links.go` (`linkedActs`)
- Modify: `services/api/internal/presentation/http/dto.go` (`mentionDTO`, campo `Mentions` em `actHitDTO`, `toHitDTO`)
- Test: `services/api/internal/integration/entity_page_test.go` (novo)

**Interfaces:**
- Consumes: `domain.EntityMention`, `domain.EntityLabel`, `domain.EntitySlug` (Task 2)
- Produces: `ActHit.Mentions` preenchido na busca e no relatório; JSON `mentions: [{kind, key, label, slug}]`, sempre array.

- [ ] **Step 1: fixture e teste de integração**

`entity_page_test.go` (tag `integration`) define a edição usada também nas Tasks 4 a 6:

```go
const processGazette = "SEMAD\nAVISO DE LICITAÇÃO\nPregão nº 4/2023. Processo Administrativo nº 2808/2022.\n" +
	"HOMOLOGAÇÃO/ADJUDICAÇÃO\nPregão nº 4/2023, Processo nº 2.808/2022. Contratada: Empresa Exemplo LTDA, CNPJ: 12.345.678/0001-90.\n" +
	"EXTRATO DO CONTRATO Nº 30/SEMAD/2023\nProcesso nº 2808/2022. Contratada: Empresa Exemplo LTDA, CNPJ: 12.345.678/0001-90. Valor global: R$ 50.000,00.\n" +
	"SEMTRAN\nEXTRATO DO CONTRATO Nº 7/SEMTRAN/2024\nProcesso nº 2808/2022. Adesão à ata do Pregão nº 4/2023.\n" +
	"EXTRATO DO PRIMEIRO TERMO ADITIVO AO CONTRATO Nº 30/SEMAD/2023\nProcesso Administrativo nº 2808/2022. Prorrogação.\n" +
	"ATOS DO PREFEITO\nDECRETO Nº 9/2024\nCita o processo nº 2808/2022 sem órgão.\n"
```

Antes de escrever as asserções, rode a fixture uma vez e confira os órgãos que o parser atribui (`SELECT title, organ FROM acts ORDER BY position`). A fixture precisa ter: dois órgãos nomeados (SEMAD e SEMTRAN), um ato sem órgão, um contrato com sigla e um aditivo. Se o parser juntar ou separar atos de outro jeito, ajuste o texto, não o parser.

Crie `newEntityServer(t)`, copiado de `newServerFor` em `investigator_test.go` mas indexando `processGazette`. Primeiro teste:

```go
func TestSearchHitsCarryProcessAndContractMentions(t *testing.T) {
	srv, _ := newEntityServer(t)
	var res struct {
		Items []struct {
			Title    string `json:"title"`
			Mentions []struct {
				Kind, Key, Label, Slug string
			} `json:"mentions"`
		} `json:"items"`
	}
	getJSON(t, srv.URL+"/v1/acts?q="+url.QueryEscape("30/SEMAD/2023"), &res)
	var found bool
	for _, it := range res.Items {
		for _, m := range it.Mentions {
			if m.Kind == "contrato" && m.Key == "30/SEMAD/2023" && m.Slug == "30-SEMAD-2023" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("esperava a menção ao contrato 30/SEMAD/2023: %+v", res.Items)
	}
}
```

Se não houver helper `getJSON` no pacote, use o que os outros testes usam (`grep -n "func getJSON\|func get(" internal/integration/*.go`) ou escreva um de 10 linhas com `http.Get` e `json.NewDecoder`.

- [ ] **Step 2: rodar e ver falhar** — `make test-integration` (ou `go test -tags integration -run TestSearchHitsCarry ./internal/integration/` com `TEST_DATABASE_URL`).

- [ ] **Step 3: implementar**

Em `search.go`:

```go
const mentionsSubquery = `(SELECT coalesce(array_agg(DISTINCT m.kind || '|' || entity_key(m.kind, m.normalized) || '|' || m.value), '{}')
		        FROM act_entities m WHERE m.act_id = a.id AND m.kind IN ('processo', 'contrato'))`

func parseMentions(raw []string) []domain.EntityMention {
	seen := map[string]bool{}
	var out []domain.EntityMention
	for _, r := range raw {
		parts := strings.SplitN(r, "|", 3)
		if len(parts) != 3 || seen[parts[0]+parts[1]] {
			continue
		}
		seen[parts[0]+parts[1]] = true
		kind := domain.EntityKind(parts[0])
		out = append(out, domain.EntityMention{Kind: kind, Key: parts[1], Label: domain.EntityLabel(kind, parts[2])})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind > out[j].Kind
		}
		return out[i].Key < out[j].Key
	})
	return out
}
```

Acrescente `mentionsSubquery` ao SELECT de `ActRepo.Search` e de `LinkRepo.linkedActs`, escaneie em `var mentions []string` com `pq.Array(&mentions)` e faça `h.Mentions = parseMentions(mentions)`. `export` e `feed` não mudam.

Em `dto.go`:

```go
type mentionDTO struct {
	Kind  string `json:"kind"`
	Key   string `json:"key"`
	Label string `json:"label"`
	Slug  string `json:"slug"`
}
```

`actHitDTO` ganha `Mentions []mentionDTO \`json:"mentions"\`` e `Phase string \`json:"phase,omitempty"\``. `toHitDTO` preenche os dois; `Mentions` nunca é `nil` (`make([]mentionDTO, 0, len(h.Mentions))`).

- [ ] **Step 4: rodar** — `make lint test` e o teste de integração → PASS.
- [ ] **Step 5: commit** — `feat(api): processos e contratos citados em cada ato`.

### Task 4: relatório de processo e contrato com órgãos, fases e citados junto

**Files:**
- Modify: `services/api/internal/adapters/postgres/links.go`, `services/api/internal/adapters/postgres/entities.go` (limites)
- Modify: `services/api/internal/core/usecase/get_entity.go`
- Create: `services/api/internal/core/usecase/get_entity_test.go` (se não existir; se existir, acrescentar)
- Test: `services/api/internal/integration/entity_page_test.go`

**Interfaces:**
- Consumes: tipos da Task 2, `PhaseOf` da Task 1, `processGazette` e `newEntityServer` da Task 3
- Produces: `EntityReport` com `Label`, `Organs` (ordenado por atos desc, órgão asc, `""` por último), `TypeTitleCounts`, `Related` e, depois do caso de uso, `CountByPhase` e `Acts[i].Phase`.

- [ ] **Step 1: teste do caso de uso (unitário, leitor falso)**

```go
type fakeEntityReader struct{ report domain.EntityReport }

func (f fakeEntityReader) ReportByKey(context.Context, domain.EntityKind, string, string) (domain.EntityReport, error) {
	return f.report, nil
}

func TestGetEntityFillsPhases(t *testing.T) {
	reader := fakeEntityReader{domain.EntityReport{
		Kind: domain.EntityProcesso,
		Acts: []domain.ActHit{{Act: domain.Act{Type: domain.ActLicitacao, Title: "HOMOLOGAÇÃO"}}},
		TypeTitleCounts: []domain.TypeTitleCount{
			{Type: domain.ActLicitacao, Title: "HOMOLOGAÇÃO", Acts: 2},
			{Type: domain.ActContrato, Title: "EXTRATO DE DISTRATO DE CONTRATO", Acts: 1},
			{Type: domain.ActDespacho, Title: "DESPACHO", Acts: 4},
		},
	}}
	got, err := usecase.NewGetEntity(reader).Execute(context.Background(), domain.EntityProcesso, "2808/2022", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Acts[0].Phase != domain.PhaseHomologacao {
		t.Fatalf("fase do ato: %s", got.Acts[0].Phase)
	}
	want := map[domain.Phase]int{domain.PhaseHomologacao: 2, domain.PhaseRescisao: 1, domain.PhaseOutro: 4}
	if !reflect.DeepEqual(got.CountByPhase, want) {
		t.Fatalf("contagem por fase: %v", got.CountByPhase)
	}
}
```

(Use o pacote de teste que os outros testes de `usecase` usam; confira com `head -5 services/api/internal/core/usecase/*_test.go`.)

- [ ] **Step 2: teste de integração do relatório**

```go
func TestProcessReportGroupsOrgansPhasesAndRelated(t *testing.T) {
	_, db := newEntityServer(t)
	report, err := usecase.NewGetEntity(postgres.NewLinkRepo(db)).
		Execute(context.Background(), domain.EntityProcesso, "2808-2022", "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Label != "2808/2022" || report.TotalActs != 6 {
		t.Fatalf("rótulo e total: %q %d", report.Label, report.TotalActs)
	}
	if len(report.Organs) != 3 || report.Organs[0].Organ != "SEMAD" || report.Organs[2].Organ != "" {
		t.Fatalf("órgãos: %+v", report.Organs)
	}
	if report.CountByPhase[domain.PhaseLicitacao] != 1 || report.CountByPhase[domain.PhaseHomologacao] != 1 ||
		report.CountByPhase[domain.PhaseContrato] != 2 || report.CountByPhase[domain.PhaseAditivo] != 1 {
		t.Fatalf("fases: %v", report.CountByPhase)
	}
	related := map[string]int{}
	for _, r := range report.Related {
		related[string(r.Kind)+":"+r.Key] = r.Acts
	}
	if related["contrato:30/SEMAD/2023"] != 2 || related["cnpj:12345678000190"] != 2 {
		t.Fatalf("citados junto: %+v", report.Related)
	}
	if _, self := related["processo:28082022"]; self {
		t.Fatal("a própria entidade não entra em citados junto")
	}
}

func TestProcessReportReturnsUpToThreeHundredActs(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= 120; i++ {
		fmt.Fprintf(&b, "DESPACHO DO SECRETÁRIO\nProcesso nº 4444/2024. Despacho %d.\n", i)
	}
	_, db := newServerFor(t, b.String())
	report, err := postgres.NewLinkRepo(db).ReportByKey(context.Background(), domain.EntityProcesso, "44442024", "")
	if err != nil || report.TotalActs != 120 || len(report.Acts) != 120 {
		t.Fatalf("esperava os 120 atos: total %d, devolvidos %d, %v", report.TotalActs, len(report.Acts), err)
	}
}
```

Ajuste os números esperados ao que a fixture de fato produz (conferido na Task 3), mantendo o que o teste quer provar: dois órgãos nomeados mais o vazio, SEMAD primeiro, fases distintas, contrato e CNPJ citados junto, sem a própria entidade.

- [ ] **Step 3: rodar e ver falhar.**

- [ ] **Step 4: implementar no `LinkRepo`**

Limites em `entities.go`: troque `reportActsLimit = 100` por uma função

```go
func reportActsLimit(kind domain.EntityKind) int {
	if kind == domain.EntityCNPJ {
		return 100
	}
	return 300
}
```

e passe `reportActsLimit(kind)` para `linkedActs` (acrescente o parâmetro `limit int`). Veja se `reportActsLimit` é usado em outro lugar (`grep -rn reportActsLimit`) e ajuste.

Em `ReportByKey`, depois de `linkedTypes`, chame (para todo tipo, inclusive CNPJ, porque o custo é uma consulta agrupada):

```go
func (r *LinkRepo) linkedOrgansAndTitles(ctx context.Context, entityID, source string, report *domain.EntityReport) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT coalesce(a.organ, ''), a.type, a.title, count(*)
		FROM entity_links l JOIN acts a ON a.id = l.record_id::uuid
		WHERE l.entity_id = $1 AND l.record_kind = $2 AND ($3 = '' OR l.source = $3)
		GROUP BY 1, 2, 3`, entityID, domain.RecordAct, source)
	if err != nil {
		return err
	}
	defer rows.Close()
	organs := map[string]int{}
	for rows.Next() {
		var organ, typ, title string
		var n int
		if err := rows.Scan(&organ, &typ, &title, &n); err != nil {
			return err
		}
		organs[organ] += n
		report.TypeTitleCounts = append(report.TypeTitleCounts, domain.TypeTitleCount{Type: domain.ActType(typ), Title: title, Acts: n})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	report.Organs = sortedOrgans(organs)
	return nil
}

func sortedOrgans(m map[string]int) []domain.OrganCount {
	out := make([]domain.OrganCount, 0, len(m))
	for o, n := range m {
		out = append(out, domain.OrganCount{Organ: o, Acts: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if (out[i].Organ == "") != (out[j].Organ == "") {
			return out[j].Organ == ""
		}
		if out[i].Acts != out[j].Acts {
			return out[i].Acts > out[j].Acts
		}
		return out[i].Organ < out[j].Organ
	})
	return out
}
```

Para processo e contrato (não para CNPJ), `linkedRelated`:

```go
const relatedLimit = 20

func (r *LinkRepo) linkedRelated(ctx context.Context, entityID, source string, report *domain.EntityReport) error {
	rows, err := r.db.QueryContext(ctx, `
		WITH acts_of AS (
			SELECT l.record_id, l.source FROM entity_links l
			WHERE l.entity_id = $1 AND l.record_kind = $2 AND ($3 = '' OR l.source = $3)
		), related AS (
			SELECT e.kind, e.key, count(DISTINCT o.record_id) AS acts,
			       row_number() OVER (PARTITION BY e.kind ORDER BY count(DISTINCT o.record_id) DESC, e.key) AS rank
			FROM acts_of o
			JOIN entity_links lr ON lr.record_kind = $2 AND lr.record_id = o.record_id AND lr.source = o.source
			JOIN entities e ON e.id = lr.entity_id AND e.id <> $1
			GROUP BY e.kind, e.key
		)
		SELECT kind, key, acts FROM related WHERE rank <= $4 ORDER BY kind, acts DESC, key`,
		entityID, domain.RecordAct, source, relatedLimit)
	...
}
```

Rótulos: uma consulta só, restrita aos atos ligados à entidade, dá o
valor escrito mais frequente de cada processo e contrato que aparece
nesses atos (a própria entidade e os relacionados):

```go
func (r *LinkRepo) labels(ctx context.Context, entityID, source string) (map[string]string, error)
```

```sql
WITH acts_of AS (
	SELECT l.record_id::uuid AS act_id FROM entity_links l
	WHERE l.entity_id = $1 AND l.record_kind = $2 AND ($3 = '' OR l.source = $3)
)
SELECT DISTINCT ON (v.kind, entity_key(v.kind, v.normalized))
       v.kind, entity_key(v.kind, v.normalized), v.value
FROM act_entities v JOIN acts_of o ON o.act_id = v.act_id
WHERE v.kind IN ('processo', 'contrato')
GROUP BY v.kind, entity_key(v.kind, v.normalized), v.value
ORDER BY v.kind, entity_key(v.kind, v.normalized), count(*) DESC, v.value
```

A chave do mapa é `kind + ":" + key`, e o valor já passa por
`domain.EntityLabel`. `ReportByKey` usa o mapa para `report.Label` e para
o rótulo de cada `Related` de processo e contrato; o de CNPJ é
`domain.EntityLabel(domain.EntityCNPJ, key)`. Meça com `EXPLAIN ANALYZE`
na base local para `2808/2022` e para o maior contrato; o alvo é ficar
abaixo de 300 ms.

- [ ] **Step 5: implementar no caso de uso**

```go
func (uc *GetEntity) Execute(ctx context.Context, kind domain.EntityKind, input, source string) (domain.EntityReport, error) {
	...
	report, err := uc.reader.ReportByKey(ctx, kind, key, source)
	if err != nil {
		return report, err
	}
	return withPhases(report), nil
}

func withPhases(r domain.EntityReport) domain.EntityReport {
	acts := make([]domain.ActHit, len(r.Acts))
	for i, h := range r.Acts {
		h.Phase = domain.PhaseOf(h.Type, h.Title)
		acts[i] = h
	}
	r.Acts = acts
	r.CountByPhase = map[domain.Phase]int{}
	for _, c := range r.TypeTitleCounts {
		r.CountByPhase[domain.PhaseOf(c.Type, c.Title)] += c.Acts
	}
	return r
}
```

- [ ] **Step 6: rodar** — `make lint test` e `make test-integration` → PASS.
- [ ] **Step 7: commit** — `feat(api): órgãos, fases e citados junto no relatório de entidade`.

### Task 5: rota HTTP de processo e contrato

**Files:**
- Modify: `services/api/internal/presentation/http/router.go` (campo `Entity *usecase.GetEntity`, rota, handler), `dto.go` (`entityResponse`)
- Modify: `services/api/cmd/api/main.go` (injetar `Entity`), `services/api/internal/integration/investigator_test.go` e `newEntityServer` (injetar `Entity`)
- Test: `services/api/internal/integration/entity_page_test.go`

**Interfaces:**
- Consumes: `GetEntity` com fases (Task 4), `EntityWarnings` (Task 2)
- Produces: `GET /v1/entities/{kind}/{key}` com o JSON abaixo; é o contrato do site (Task 7).

```go
type organCountDTO struct {
	Organ     string `json:"organ"`
	OrganName string `json:"organ_name"`
	Acts      int    `json:"acts"`
}

type relatedDTO struct {
	Kind  string `json:"kind"`
	Key   string `json:"key"`
	Label string `json:"label"`
	Slug  string `json:"slug"`
	Acts  int    `json:"acts"`
}

type entityResponse struct {
	Kind         string          `json:"kind"`
	Key          string          `json:"key"`
	Label        string          `json:"label"`
	Certainty    string          `json:"certainty"`
	Diarios      int             `json:"diarios"`
	TotalActs    int             `json:"total_acts"`
	CountByPhase map[string]int  `json:"count_by_phase"`
	Organs       []organCountDTO `json:"organs"`
	Related      []relatedDTO    `json:"related"`
	Warnings     []string        `json:"warnings"`
	Acts         []actHitDTO     `json:"acts"`
}
```

`Label` vazio (número sem nenhum ato) cai para o que a pessoa digitou, com `-` trocado por `/` quando for contrato.

- [ ] **Step 1: teste de integração**

```go
func TestEntityRouteServesProcessAndContract(t *testing.T) {
	srv, _ := newEntityServer(t)
	var p struct {
		Label        string         `json:"label"`
		TotalActs    int            `json:"total_acts"`
		CountByPhase map[string]int `json:"count_by_phase"`
		Organs       []struct{ Organ string } `json:"organs"`
		Warnings     []string       `json:"warnings"`
		Acts         []struct{ Phase string } `json:"acts"`
	}
	getJSON(t, srv.URL+"/v1/entities/processo/2808-2022", &p)
	if p.Label != "2808/2022" || p.TotalActs != 6 || len(p.Organs) != 3 || len(p.Warnings) != 1 || p.Acts[0].Phase == "" {
		t.Fatalf("processo: %+v", p)
	}
	var c struct{ Key string; TotalActs int `json:"total_acts"` }
	getJSON(t, srv.URL+"/v1/entities/contrato/30-SEMAD-2023", &c)
	if c.Key != "30/SEMAD/2023" || c.TotalActs != 2 {
		t.Fatalf("contrato: %+v", c)
	}
	for path, status := range map[string]int{
		"/v1/entities/valor/100":           http.StatusNotFound,
		"/v1/entities/processo/12":         http.StatusBadRequest,
		"/v1/entities/processo/99999-2001": http.StatusOK,
	} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != status {
			t.Errorf("%s: esperava %d, veio %d", path, status, res.StatusCode)
		}
	}
}
```

A rota de CNPJ (`GET /v1/entities/cnpj/{cnpj}`) é mais específica e continua com o handler antigo; confira que o teste existente dela segue passando.

- [ ] **Step 2: rodar e ver falhar.**
- [ ] **Step 3: implementar**

```go
mux.HandleFunc("GET /v1/entities/{kind}/{key}", a.getEntity)

func (a *API) getEntity(w http.ResponseWriter, r *http.Request) {
	kind := domain.EntityKind(r.PathValue("kind"))
	if kind != domain.EntityProcesso && kind != domain.EntityContrato {
		writeError(w, fmt.Errorf("%w: tipo de entidade %q", domain.ErrNotFound, kind), a.Log)
		return
	}
	report, err := a.Entity.Execute(r.Context(), kind, r.PathValue("key"), "")
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	writeJSON(w, http.StatusOK, toEntityResponse(report, r.PathValue("key")))
}
```

`toEntityResponse` monta o DTO com `domain.OrganName` para `organ_name`, `domain.EntitySlug(label)` para `slug`, `domain.EntityWarnings(report)` (nunca `nil`) e `toHitDTO` com `Phase: string(h.Phase)`.

- [ ] **Step 4: rodar** — `make lint test`, `make test-integration` → PASS.
- [ ] **Step 5: commit** — `feat(api): rota de processo e contrato`.

### Task 6: `entidade` do MCP com fase, órgãos e citados junto

**Files:**
- Modify: `services/api/internal/presentation/mcp/tools.go` (`entityOutput`, `entity`), `dto.go` (`actSummaryDTO.Phase`)
- Test: `services/api/internal/integration/entity_page_test.go` (ou `mcp_test.go`, seguindo o helper `call` que já existe lá)

**Interfaces:**
- Consumes: relatório da Task 4, `EntityWarnings` da Task 2
- Produces: em `entityOutput`, `Label string \`json:"rotulo,omitempty"\``, `CountByPhase map[string]int \`json:"atos_por_fase,omitempty"\``, `Organs []organDTO \`json:"orgaos,omitempty"\`` (`orgao`, `orgao_nome`, `atos`), `Related []relatedEntityDTO \`json:"citados_junto,omitempty"\`` (`tipo`, `chave`, `rotulo`, `atos`); em `actSummaryDTO`, `Phase string \`json:"fase,omitempty"\``.

- [ ] **Step 1: teste** — pelo cliente MCP, `entidade` com `{"tipo": "processo", "numero": "2808/2022"}` sobre `processGazette`: `rotulo` = `2808/2022`, `atos_por_fase.contrato` = 2, `orgaos` com 3 itens, `citados_junto` com o contrato `30/SEMAD/2023`, `aviso` contendo "aparece em 2 órgãos" e `atos_recentes[0].fase` não vazio. Para CNPJ, `orgaos` vem preenchido e `citados_junto` vazio.
- [ ] **Step 2: rodar e ver falhar.**
- [ ] **Step 3: implementar** — preencher os campos em `entity`; `summaryOf` copia `string(h.Phase)`. `atos_por_fase` e `citados_junto` saem só para processo e contrato.
- [ ] **Step 4: rodar** — `make lint test`, `make test-integration` → PASS.
- [ ] **Step 5: commit** — `feat(mcp): entidade com fase, órgãos e citados junto`.

### Task 7: páginas no site

**Files:**
- Modify: `apps/web/src/api.ts` (tipos e `getEntity`), `apps/web/src/types.ts` (`PHASE_LABEL`, `PHASES`), `apps/web/src/App.tsx` (rotas), `apps/web/src/components.tsx` (`Result` com fase e menções), `apps/web/src/styles.css`
- Create: `apps/web/src/entity.ts`, `apps/web/src/entity.test.ts`, `apps/web/src/EntityPage.tsx`

**Interfaces:**
- Consumes: JSON da Task 5 e `mentions` da Task 3
- Produces:
  - `api.ts`: `type Phase`, `interface Mention { kind: "processo" | "contrato"; key: string; label: string; slug: string }`, `ActHit.mentions: Mention[]`, `ActHit.phase?: Phase`, `interface EntityResponse` (espelho de `entityResponse`), `getEntity(kind, slug)`
  - `entity.ts`: `entityPath(kind, slug): string`, `parseEntityPath(pathname): { kind: "processo" | "contrato"; slug: string } | null`, `groupByOrgan(acts: ActHit[], organs: OrganCount[], selected: string): { organ: string; acts: ActHit[] }[]`

- [ ] **Step 1: testes em `entity.test.ts`**

```ts
import { describe, expect, it } from "vitest";
import { ActHit } from "./api";
import { entityPath, groupByOrgan, parseEntityPath } from "./entity";

const hit = (id: string, organ: string, published_at: string) => ({ id, organ, published_at }) as ActHit;

describe("entityPath e parseEntityPath", () => {
  it("vão e voltam pelo slug", () => {
    expect(entityPath("contrato", "30-FMS-2011")).toBe("/contrato/30-FMS-2011");
    expect(parseEntityPath("/contrato/30-FMS-2011")).toEqual({ kind: "contrato", slug: "30-FMS-2011" });
    expect(parseEntityPath("/processo/14.672-2021")).toEqual({ kind: "processo", slug: "14.672-2021" });
    expect(parseEntityPath("/empresa/123")).toBeNull();
  });
});

describe("groupByOrgan", () => {
  const organs = [{ organ: "SEMAD", organ_name: "", acts: 2 }, { organ: "SEMTRAN", organ_name: "", acts: 1 }, { organ: "", organ_name: "", acts: 1 }];
  const acts = [hit("c", "SEMAD", "2024-03-27"), hit("d", "", "2024-05-01"), hit("b", "SEMTRAN", "2024-02-02"), hit("a", "SEMAD", "2023-02-27")];

  it("agrupa na ordem dos órgãos, sem órgão por último, do mais antigo ao mais recente", () => {
    expect(groupByOrgan(acts, organs, "").map((g) => [g.organ, g.acts.map((a) => a.id)])).toEqual([
      ["SEMAD", ["a", "c"]], ["SEMTRAN", ["b"]], ["", ["d"]],
    ]);
  });

  it("filtra pelo órgão escolhido", () => {
    expect(groupByOrgan(acts, organs, "SEMTRAN").map((g) => g.organ)).toEqual(["SEMTRAN"]);
  });
});
```

- [ ] **Step 2: rodar e ver falhar** — `cd apps/web && npm test`.

- [ ] **Step 3: implementar `entity.ts`**

```ts
import { ActHit, OrganCount } from "./api";

export type EntityKind = "processo" | "contrato";

export function entityPath(kind: EntityKind, slug: string) {
  return `/${kind}/${slug}`;
}

export function parseEntityPath(pathname: string): { kind: EntityKind; slug: string } | null {
  const m = pathname.match(/^\/(processo|contrato)\/([^/]+)$/);
  return m ? { kind: m[1] as EntityKind, slug: decodeURIComponent(m[2]) } : null;
}

export function groupByOrgan(acts: ActHit[], organs: OrganCount[], selected: string) {
  const byDate = [...acts].sort((a, b) => a.published_at.localeCompare(b.published_at) || a.id.localeCompare(b.id));
  return organs
    .filter((o) => selected === "" || o.organ === selected)
    .map((o) => ({ organ: o.organ, acts: byDate.filter((a) => (a.organ || "") === o.organ) }))
    .filter((g) => g.acts.length > 0);
}
```

- [ ] **Step 4: `EntityPage.tsx`**, seguindo `CompanyPage.tsx`:
  - carrega `getEntity(kind, slug)`; mostra "Carregando…" e erro como a página da empresa;
  - `<p className="eyebrow">Processo</p>` ou `Contrato`, `<h1>{data.label}</h1>`;
  - `data.warnings` como `<p className="notice">` cada;
  - `<dl className="summary">` com "Atos" (`total_acts`) e uma entrada por fase de `PHASES` que tenha contagem, com `PHASE_LABEL`;
  - se `total_acts > acts.length`: `<p className="fineprint">Mostrando os {acts.length} atos mais recentes de {total_acts}.</p>`;
  - com mais de um item em `organs`, botões de filtro (`Todos` e um por órgão com a contagem; o vazio se chama "Sem órgão"), com o escolhido em `?orgao=` via `history.replaceState`;
  - para cada grupo de `groupByOrgan`, `<section>` com `<h2>` do órgão (sigla e nome, ou "Sem órgão", omitido quando há um grupo só) e `<ol className="timeline">` com `<Result hit={h} />`;
  - "Citados junto": lista de `related`, com link `entityPath(kind, slug)` para processo e contrato e `/empresa/{key}` para CNPJ, e o número de atos;
  - sem atos: `<p className="count">Nenhum ato indexado cita este número.</p>`.
- [ ] **Step 5: `App.tsx`** — antes da rota da empresa: `const entity = parseEntityPath(path); if (entity) return <EntityPage kind={entity.kind} slug={entity.slug} />;`.
- [ ] **Step 6: `Result`** — com `hit.phase`, um `<span className="phase">{PHASE_LABEL[hit.phase]}</span>` depois da etiqueta de tipo; com `hit.mentions.length > 0`, um `<p className="mentions">Processos e contratos citados: …</p>` com links `entityPath(m.kind, m.slug)` e o texto `m.label`, no mesmo formato de "Empresas citadas".
- [ ] **Step 7: `types.ts`** — `PHASE_LABEL: Record<Phase, string>`: licitação "Licitação", homologação "Homologação", ata "Ata de registro de preços", dispensa "Dispensa ou inexigibilidade", contrato "Contrato", fiscal "Fiscal do contrato", aditivo "Aditivo ou apostilamento", ajuste de contas "Ajuste de contas", rescisão "Rescisão ou distrato", outro "Outros atos"; `PHASES` na ordem de `domain.Phases`.
- [ ] **Step 8: estilos** — `.phase`, `.mentions`, `.organ-filter` reaproveitando as variáveis e o visual de `.tag`, `.cnpjs` e dos botões de tipo da busca.
- [ ] **Step 9: rodar** — `cd apps/web && npm run typecheck && npm test` → PASS.
- [ ] **Step 10: verificação no navegador** — `make run-api` e `make run-web`, abrir `/processo/2808-2022`, `/contrato/30-SEMAD-2023` (ou um contrato real com aditivos e distrato), um processo de um ato só e uma busca por `contrato`; conferir fases contra os títulos, filtro por órgão e links das menções. Sem a extensão do Chrome, usar Playwright headless e capturar tela.
- [ ] **Step 11: commit** — `feat(web): páginas de processo e de contrato`.

### Task 8: documentação e verificação final

**Files:** `README.md` (rota nova, campos `mentions` e `phase`, campos novos da `entidade`), `docs/roadmap.md` (Entrega 2: páginas de processo e contrato entregues, com o commit), `docs/decisoes-de-codigo.md` (fase calculada na leitura e por que; órgão fora da chave e o aviso de vários órgãos).

- [ ] Medir na base local o tempo de `GET /v1/entities/processo/2808-2022` e de um CNPJ grande (antes e depois da mudança) e registrar em `docs/decisoes-de-codigo.md` se o CNPJ ficar mais lento que 1 s.
- [ ] `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.
- [ ] Commit `docs(roadmap): páginas de processo e contrato`.
