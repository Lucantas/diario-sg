# Controle do TCE-RJ Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Carregar pareceres, débitos e multas e obras paralisadas do TCE-RJ sobre São Gonçalo e mostrá-los em `/tce` e na página da empresa.

**Architecture:** O job `tce` ganha um terceiro caso de uso, que lê três conjuntos da API do TCE, filtra São Gonçalo, arquiva as respostas e troca três tabelas numa transação, ligando as obras às empresas pelo CNPJ. A leitura monta a página `/tce` e a seção da empresa.

**Tech Stack:** Go, Postgres, React.

**Spec:** `docs/superpowers/specs/2026-09-26-controle-do-tce-design.md`

## Global Constraints

- Sem comentários no código; conventional commits em português, sem link de sessão; commits como Lucas Dantas <lucas.lucantas38@gmail.com>.
- Migration nova: `019_tce_oversight.sql`.
- Fonte `tce_controle`; mesma base do TCE (`TCE_BASE_URL`).
- `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npx vitest run`.
- Nada rodando ao final.

---

### Task 1: Modelo, leitura e carga

**Files:** Create `migrations/019_tce_oversight.sql`, `domain/tce_oversight.go` (+ teste), `usecase/load_oversight.go` (+ teste), `postgres/oversight.go`, `integration/oversight_test.go`. Modify `adapters/tce/source.go` (+ teste), `ports.go`, `cmd/tce/main.go`, `resetTables`.

**Interfaces (produz):**
- `domain.TCEAccount`, `domain.TCEPenalty`, `domain.StalledWork`, `domain.TCEOversight{Accounts, Penalties, Works}`
- `domain.ParseTCEAccounts/ParseTCEPenalties/ParseStalledWorks(body []byte) (…, total int, err error)` (só São Gonçalo)
- `tce.(*Source).Dataset(ctx, name string) ([]byte, error)`
- `ports.OversightRepository { Ready(ctx) error; ReplaceOversight(ctx, o domain.TCEOversight) error }`
- `usecase.NewLoadOversight(src, repo, runs, raw, now).Execute(ctx) (domain.FetchRun, error)`

- [ ] Testes; migration; commit `feat(api): carga de contas, multas e obras paralisadas do TCE-RJ`.

### Task 2: Página `/tce` e empresa

**Files:** Create `usecase/get_oversight.go`, `presentation/http/oversight.go`, `apps/web/src/TCEPage.tsx`, `tce.ts` (+ teste), `StalledWorksSection.tsx`. Modify `ports.go` (`OversightReader`), `postgres/oversight.go`, `domain/entity.go` (`StalledWorks` em `CompanyFacts`), `company_sources.go`, `router.go`, `dto.go`, MCP, `cmd/api/main.go`, `App.tsx`, `SearchPage.tsx`, `CompanyPage.tsx`, `api.ts`, integração.

- [ ] `GET /v1/tce`; `stalled_works` na empresa; MCP `obras_paralisadas_tce`.
- [ ] Página com as três seções e busca do processo no Diário.
- [ ] Commit `feat: contas, multas e obras paralisadas do TCE-RJ`.

### Task 3: Carga real e docs

- [ ] Carga real; conferir contra o JSON; Playwright de `/tce` e da empresa em 1280 e 390 px.
- [ ] Roadmap, README, spec "Depois da entrega"; commit `docs(roadmap): controle do TCE-RJ`.
