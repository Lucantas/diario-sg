# Busca de investigador — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Faixa de valor, operador `OU`, valores citados em cada ato, paginação, período e URL permanente para cada busca.

**Architecture:** `ActFilter` ganha `MinCents`/`MaxCents` e passa a consulta por `TranslateOperators`. O adapter do Postgres ganha `filterSQL`, usado por `Search` e `CountByMonth`. O front move a busca para `SearchPage.tsx` e guarda o estado na URL via `src/searchState.ts`.

**Tech Stack:** Go 1.23, PostgreSQL 16 (`websearch_to_tsquery`), React 18, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-22-busca-de-investigador-design.md`

## Global Constraints

- Zero comentários novos no código.
- Sem migration.
- API em reais com ponto decimal (`^\d+(\.\d{1,2})?$`); o front converte o formato brasileiro.
- `OU` só maiúsculo e fora de aspas vira `or`.
- Conventional commits em português, sem link de sessão de agente.
- Antes de concluir: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.

---

### Task 1: Operadores e faixa de valor no domínio

**Files:**
- Create: `services/api/internal/core/domain/query.go`, `query_test.go`
- Modify: `services/api/internal/core/domain/act.go` (`ActFilter`, `ActHit`)
- Modify: `services/api/internal/core/usecase/match_subscriptions.go`

**Interfaces:**
- Produces: `domain.TranslateOperators(string) string`; `ActFilter.MinCents`, `ActFilter.MaxCents int64`; `ActHit.ValuesCents []int64`.

- [ ] **Step 1: Testes que falham**

```go
func TestTranslateOperators(t *testing.T) {
	cases := map[string]string{
		"limpeza OU coleta":             "limpeza or coleta",
		"limpeza ou coleta":             "limpeza ou coleta",
		"OUTORGA":                       "OUTORGA",
		`"carne OU frango" OU peixe`:    `"carne OU frango" or peixe`,
		"a OU b OU c":                   "a or b or c",
		"":                              "",
	}
	for in, want := range cases {
		if got := TranslateOperators(in); got != want {
			t.Errorf("TranslateOperators(%q) = %q, esperava %q", in, got, want)
		}
	}
}

func TestFilterValueRange(t *testing.T) {
	for _, f := range []ActFilter{{MinCents: -1}, {MaxCents: -1}, {MinCents: 500, MaxCents: 100}} {
		if err := f.Normalize(); !errors.Is(err, ErrInvalidFilter) {
			t.Errorf("%+v: esperava ErrInvalidFilter, veio %v", f, err)
		}
	}
	f := ActFilter{MinCents: 100, MaxCents: 100, Query: "a OU b"}
	if err := f.Normalize(); err != nil || f.Query != "a or b" {
		t.Errorf("veio %+v %v", f, err)
	}
}
```

- [ ] **Step 2:** `go test ./internal/core/domain/` → FAIL.
- [ ] **Step 3: Implementação**

```go
var orWord = regexp.MustCompile(`(^|\s)OU(\s|$)`)

func TranslateOperators(q string) string {
	parts := strings.Split(q, `"`)
	for i := 0; i < len(parts); i += 2 {
		for orWord.MatchString(parts[i]) {
			parts[i] = orWord.ReplaceAllString(parts[i], "${1}or${2}")
		}
	}
	return strings.Join(parts, `"`)
}
```

(`for` porque `a OU b OU c` tem ocorrências que compartilham o espaço.)
`Normalize`: validar faixa e `f.Query = TranslateOperators(f.Query)`.
`MatchSubscriptions`: `SearchInGazette(ctx, g.ID, domain.TranslateOperators(s.Query))`.

- [ ] **Step 4:** `go test ./internal/core/...` → PASS.
- [ ] **Step 5:** commit `feat(domain): operador OU e faixa de valor no filtro`.

### Task 2: Faixa de valor e valores citados no Postgres e na API

**Files:**
- Create: `services/api/internal/adapters/postgres/filter.go`
- Modify: `services/api/internal/adapters/postgres/acts.go` (`Search`), `entities.go` (`CountByMonth`)
- Create: `services/api/internal/presentation/http/money.go`, `money_test.go`
- Modify: `services/api/internal/presentation/http/router.go` (`filterFromQuery`), `dto.go`
- Create: `services/api/internal/integration/investigator_test.go`

**Interfaces:**
- Consumes: `ActFilter.MinCents/MaxCents`, `ActHit.ValuesCents`.
- Produces: `filterSQL(f domain.ActFilter, next int) (string, []any)`; `parseReais(string) (int64, error)`; JSON `values_cents` em `actHitDTO`; parâmetros `min_value`, `max_value`.

- [ ] **Step 1: Testes que falham**

```go
func TestParseReais(t *testing.T) {
	ok := map[string]int64{"": 0, "1500": 150000, "1500.5": 150050, "1500.50": 150050, "0.01": 1}
	for in, want := range ok {
		if got, err := parseReais(in); err != nil || got != want {
			t.Errorf("parseReais(%q) = %d, %v; esperava %d", in, got, err, want)
		}
	}
	for _, in := range []string{"1.500,50", "1,5", "-3", "abc", "1.505", "1e3"} {
		if _, err := parseReais(in); err == nil {
			t.Errorf("parseReais(%q) deveria falhar", in)
		}
	}
}
```

Integração (`investigator_test.go`), edição:

```go
const investigatorGazette = "EXTRATO DO CONTRATO Nº 1/2026\nObjeto: limpeza urbana. Valor: R$ 1.000,00.\n" +
	"EXTRATO DO CONTRATO Nº 2/2026\nObjeto: coleta de lixo hospitalar. Valor: R$ 50.000,00.\n" +
	"EXTRATO DO CONTRATO Nº 3/2026\nObjeto: limpeza de escolas. Valor global R$ 200.000,00, mensal R$ 20.000,00."
```

Conferências: `min_value=40000` → contratos 2 e 3; `max_value=10000` → 1; `min_value=15000&max_value=30000` → 3 (casa pelo mensal); `values_cents` do 3 = `[20000000, 2000000]`; `q=limpeza OU coleta` → 3 atos; `q=limpeza -urbana` → só o 3; `/v1/stats/acts?min_value=40000` → 2; `min_value=abc` → 400; `min_value=500&max_value=100` → 400.

- [ ] **Step 2:** `go test ./internal/presentation/http/` e `make test-integration` → FAIL.
- [ ] **Step 3: Implementação**

```go
var reaisRe = regexp.MustCompile(`^(\d+)(?:\.(\d{1,2}))?$`)

func parseReais(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	m := reaisRe.FindStringSubmatch(s)
	if m == nil || len(m[1]) > 13 {
		return 0, domain.ErrInvalidFilter
	}
	whole, _ := strconv.ParseInt(m[1], 10, 64)
	cents, _ := strconv.ParseInt((m[2] + "00")[:2], 10, 64)
	return whole*100 + cents, nil
}
```

```go
func filterSQL(f domain.ActFilter, next int) (string, []any) {
	var b strings.Builder
	var args []any
	param := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(next+len(args)-1)
	}
	if f.Type != "" {
		b.WriteString(" AND a.type = " + param(string(f.Type)))
	}
	if f.Organ != "" {
		b.WriteString(" AND a.organ = " + param(f.Organ))
	}
	if !f.From.IsZero() {
		b.WriteString(" AND g.published_at >= " + param(f.From.Format(time.DateOnly)) + "::date")
	}
	if !f.To.IsZero() {
		b.WriteString(" AND g.published_at <= " + param(f.To.Format(time.DateOnly)) + "::date")
	}
	if f.MinCents > 0 || f.MaxCents > 0 {
		b.WriteString(" AND EXISTS (SELECT 1 FROM act_entities v WHERE v.act_id = a.id AND v.kind = 'valor'")
		if f.MinCents > 0 {
			b.WriteString(" AND v.normalized::bigint >= " + param(f.MinCents))
		}
		if f.MaxCents > 0 {
			b.WriteString(" AND v.normalized::bigint <= " + param(f.MaxCents))
		}
		b.WriteString(")")
	}
	return b.String(), args
}
```

`Search`: parâmetros fixos `$1` query, `$2` limit, `$3` offset, `$4` like; `where, extra := filterSQL(f, 5)`; `args := append([]any{f.Query, f.Limit, f.Offset, likePattern(f.Query)}, extra...)`; seleciona
`(SELECT coalesce(array_agg(v.normalized::bigint ORDER BY v.normalized::bigint DESC), '{}') FROM act_entities v WHERE v.act_id = a.id AND v.kind = 'valor')` com `pq.Array(&h.ValuesCents)`.
`CountByMonth`: `$1` query, `$2` like, `filterSQL(f, 3)`.
Teste unitário de `filterSQL` em `search_clause_test.go` (numeração dos parâmetros e cláusulas presentes para filtro vazio, só órgão e faixa completa).
`filterFromQuery`: `f.MinCents, err = parseReais(q.Get("min_value"))` (idem max; erro → 400).
DTO: `ValuesCents []int64 json:"values_cents"` (nunca `null`).

- [ ] **Step 4:** `make lint test test-integration` → PASS.
- [ ] **Step 5:** commit `feat(api): faixa de valor e valores citados na busca`.

### Task 3: Estado da busca na URL

**Files:**
- Create: `apps/web/src/searchState.ts`, `searchState.test.ts`

**Interfaces:**
- Produces:

```ts
export interface SearchState {
  q: string; type: ActType | ""; organ: string; from: string; to: string; min: string; max: string; page: number;
}
export const EMPTY_STATE: SearchState;
export function stateFromQuery(search: string): SearchState;
export function queryFromState(s: SearchState): string;
export function parseBRL(text: string): string | null;
export function apiParams(s: SearchState, limit: number): URLSearchParams | null;
export function hasSearch(s: SearchState): boolean;
```

`min`/`max` guardam o texto digitado; `apiParams` devolve `null` se algum não for válido por `parseBRL`. `hasSearch` é verdadeiro com termo ou qualquer filtro.

- [ ] **Step 1: Testes que falham**

```ts
describe("parseBRL", () => {
  it("aceita formato brasileiro e inteiro", () => {
    expect(parseBRL("1.500,50")).toBe("1500.50");
    expect(parseBRL("1500")).toBe("1500");
    expect(parseBRL("1500,5")).toBe("1500.5");
    expect(parseBRL(" R$ 20.000 ")).toBe("20000");
  });
  it("recusa o que não é valor", () => {
    expect(parseBRL("")).toBeNull();
    expect(parseBRL("abc")).toBeNull();
    expect(parseBRL("1,2,3")).toBeNull();
    expect(parseBRL("-5")).toBeNull();
  });
});

describe("URL da busca", () => {
  it("vai e volta sem perder filtros", () => {
    const s = { q: "limpeza OU coleta", type: "contrato", organ: "SEMED", from: "2024-01-01",
      to: "2024-12-31", min: "1.000,00", max: "", page: 3 } as const;
    expect(stateFromQuery(queryFromState(s))).toEqual(s);
  });
  it("usa nomes em português e omite o vazio", () => {
    expect(queryFromState({ ...EMPTY_STATE, q: "merenda", organ: "SEMED" })).toBe("?q=merenda&orgao=SEMED");
  });
  it("ignora página inválida e tipo desconhecido", () => {
    expect(stateFromQuery("?pagina=-2&tipo=bobagem")).toEqual(EMPTY_STATE);
  });
});

describe("apiParams", () => {
  it("converte valores e página para a API", () => {
    const p = apiParams({ ...EMPTY_STATE, q: "x", min: "1.500,50", page: 2 }, 20)!;
    expect(p.get("min_value")).toBe("1500.50");
    expect(p.get("offset")).toBe("20");
  });
  it("recusa valor inválido", () => {
    expect(apiParams({ ...EMPTY_STATE, max: "abc" }, 20)).toBeNull();
  });
});
```

- [ ] **Step 2:** `npm test` → FAIL.
- [ ] **Step 3: Implementação**

```ts
import { ActType } from "./api";
import { TYPE_LABEL } from "./types";

export interface SearchState {
  q: string; type: ActType | ""; organ: string; from: string; to: string; min: string; max: string; page: number;
}

export const EMPTY_STATE: SearchState = { q: "", type: "", organ: "", from: "", to: "", min: "", max: "", page: 1 };

const KEYS = { q: "q", type: "tipo", organ: "orgao", from: "de", to: "ate", min: "valor_min", max: "valor_max" } as const;

export function stateFromQuery(search: string): SearchState {
  const p = new URLSearchParams(search);
  const type = p.get(KEYS.type) ?? "";
  const page = Number(p.get("pagina"));
  return {
    q: p.get(KEYS.q) ?? "",
    type: type in TYPE_LABEL ? (type as ActType) : "",
    organ: p.get(KEYS.organ) ?? "",
    from: p.get(KEYS.from) ?? "",
    to: p.get(KEYS.to) ?? "",
    min: p.get(KEYS.min) ?? "",
    max: p.get(KEYS.max) ?? "",
    page: Number.isInteger(page) && page > 1 ? page : 1,
  };
}

export function queryFromState(s: SearchState): string {
  const p = new URLSearchParams();
  for (const [field, key] of Object.entries(KEYS) as [keyof typeof KEYS, string][]) {
    if (s[field]) p.set(key, s[field]);
  }
  if (s.page > 1) p.set("pagina", String(s.page));
  const text = p.toString();
  return text ? `?${text}` : "";
}

export function parseBRL(text: string): string | null {
  const t = text.replace(/R\$/i, "").trim();
  if (!/^\d{1,3}(\.\d{3})*(,\d{1,2})?$|^\d+(,\d{1,2})?$/.test(t)) return null;
  return t.replace(/\./g, "").replace(",", ".");
}

export function hasSearch(s: SearchState) {
  return Boolean(s.q || s.type || s.organ || s.from || s.to || s.min || s.max);
}

export function apiParams(s: SearchState, limit: number): URLSearchParams | null {
  const p = new URLSearchParams({ q: s.q, limit: String(limit), offset: String((s.page - 1) * limit) });
  if (s.type) p.set("type", s.type);
  if (s.organ) p.set("organ", s.organ);
  if (s.from) p.set("from", s.from);
  if (s.to) p.set("to", s.to);
  for (const [field, key] of [["min", "min_value"], ["max", "max_value"]] as const) {
    if (!s[field]) continue;
    const value = parseBRL(s[field]);
    if (value === null) return null;
    p.set(key, value);
  }
  return p;
}
```

- [ ] **Step 4:** `npm test && npm run typecheck` → PASS.
- [ ] **Step 5:** commit `feat(web): estado da busca na URL`.

### Task 4: Página de busca com filtros, paginação e valores

**Files:**
- Create: `apps/web/src/SearchPage.tsx` (sai de `App.tsx`, com `AlertForm`)
- Modify: `apps/web/src/App.tsx`, `api.ts` (`searchActs(params: URLSearchParams)`, `values_cents`), `components.tsx` (valores citados), `styles.css`

- [ ] **Step 1:** `searchActs(params)` → `request<SearchResponse>(`/v1/acts?${params}`)`.
- [ ] **Step 2:** `SearchPage`: estado inicial `stateFromQuery(location.search)`; `useEffect` na montagem busca se `hasSearch`; `popstate` restaura; `submit` e mudança de filtro fazem `history.pushState(null, "", "/" + queryFromState(next))` e buscam com página 1; paginação muda só `page` e rola para o topo dos resultados. Erro de valor inválido aparece em `notice-error` sem chamar a API.
- [ ] **Step 3:** "Mais filtros" em `<details>` (aberto quando `from/to/min/max` preenchidos): dois `input type=date` e dois `input inputMode=decimal`, com rótulos visíveis e botão "Aplicar filtros"; "Limpar filtros" volta a `EMPTY_STATE` mantendo o termo.
- [ ] **Step 4:** Paginação com `nav aria-label="Páginas"`: "Anterior" (desabilitado na 1), "Página N de M", "Próxima" (desabilitado na última).
- [ ] **Step 5:** `Result`: se `values_cents.length`, `<p className="values">Valores citados: …</p>` com `formatCents` dos 3 primeiros e `+N`.
- [ ] **Step 6:** dica: `Entre aspas ("josé da silva") só a frase exata. OU junta termos (merenda OU alimentação); -termo exclui (limpeza -urbana).`
- [ ] **Step 7:** `npm run typecheck && npm test`; conferir na base local: URL com todos os filtros abre preenchida; voltar do navegador; paginação.
- [ ] **Step 8:** commit `feat(web): filtros de período e valor, paginação e URL permanente`.

### Task 5: Documentação

- [ ] README (parâmetros `min_value`/`max_value`, `values_cents`, operadores), decisões (faixa casa com qualquer valor citado, `OU` maiúsculo, reais com ponto na API), roadmap. Commit `docs: busca de investigador`.
