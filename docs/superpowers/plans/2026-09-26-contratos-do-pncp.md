# Contratos do PNCP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Carregar toda semana os contratos do município no PNCP, mostrá-los na página da empresa e no MCP e listar os que não têm extrato no Diário.

**Architecture:** Um job do módulo da API (`cmd/pncp`) pede à API de consulta do PNCP os contratos de cada órgão municipal por ano, descarta pessoa física, arquiva o que leu e troca os anos lidos em `pncp_contracts` numa transação, refazendo as ligações. A página da empresa lê pelo CNPJ; o padrão cruza com as entidades do Diário.

**Tech Stack:** Go (`net/http`, `encoding/json`), Postgres, React, Terraform.

**Spec:** `docs/superpowers/specs/2026-09-26-contratos-do-pncp-design.md`

## Global Constraints

- Sem comentários no código; conventional commits em português, sem link de sessão; commits como Lucas Dantas <lucas.lucantas38@gmail.com>.
- Migration nova: `017_pncp_contracts.sql`.
- Fonte `pncp_contratos`; base `https://pncp.gov.br/api/consulta/v1/` (`PNCP_BASE_URL`).
- Nenhum contrato de pessoa física chega ao banco ou ao bucket.
- `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npx vitest run`.
- Nada rodando ao final.

---

### Task 1: Modelo, leitura e carga

**Files:** Create `migrations/017_pncp_contracts.sql`, `domain/pncp.go`, `adapters/pncp/source.go` (+ teste), `usecase/load_pncp.go` (+ teste), `adapters/postgres/pncp.go`, `cmd/pncp/main.go`, `integration/pncp_test.go`. Modify `ports.go`, `config.go` (`RolePNCP`, `PNCPBaseURL`), `Makefile`, `resetTables`.

**Interfaces (produz):**
- `domain.PNCPContract`, `(PNCPContract).URL()`, `domain.MunicipalOrgCNPJs()`
- `pncp.New(baseURL string, client *http.Client) *Source`; `Contracts(ctx, org string, year int, each func(domain.PNCPContract, bool) error) (string, error)`
- `ports.PNCPRepository { Ready(ctx) error; ReplaceYears(ctx, from, to int, contracts []domain.PNCPContract) error }`
- `ports.PNCPReader { PNCPContractsBySupplier(ctx, cnpj string) ([]domain.PNCPContract, error) }`
- `usecase.NewLoadPNCP(src, repo, runs, raw, now).Execute(ctx, from, to int) (domain.FetchRun, error)`

- [ ] Testes do adapter e do caso de uso com fakes.
- [ ] Repositório: troca por anos e ligações `exata`; integração com duas cargas de anos diferentes.
- [ ] Commit `feat(api): carga dos contratos do PNCP`.

### Task 2: Página da empresa, MCP e padrão

**Files:** Modify `domain/entity.go` (`PNCPContracts` em `CompanyFacts`), `usecase/company_sources.go`, `domain/pattern.go`, create `domain/pncp_patterns.go` (+ teste), `usecase/list_patterns.go`, `postgres/supplier_patterns.go` (`AllPNCPContracts`, `CitedProcesses`, `LatestGazetteDay`), `presentation/http`, `presentation/mcp`, `cmd/api/main.go`; `apps/web/src/api.ts`, `PNCPSection.tsx`, `CompanyPage.tsx`, `PatternsPage.tsx`.

**Interfaces (produz):**
- `domain.FindPNCPWithoutExtract(contracts []PNCPContract, cited, citedProcesses map[string]bool, lastDiario time.Time) []PNCPContract`
- `domain.Finding.Link *FindingLink{Label, URL string}`; JSON `link`.

- [ ] Teste do padrão: citado pelo CNPJ ou pelo processo não entra; abaixo de R$ 100 mil e assinado a menos de 30 dias da última edição não entram.
- [ ] JSON `pncp_contracts`; MCP `contratos_pncp`; seção na página; link no caso do padrão.
- [ ] Commit `feat: contratos do PNCP na empresa, no MCP e nos padrões`.

### Task 3: Nuvem, carga real e docs

- [ ] Terraform: job `pncp`, agendamento `0 6 * * 0`, conta de serviço; Dockerfile e deploy.
- [ ] Carga real local; conferir 5 contratos no PNCP; Playwright da página de uma empresa e de `/padroes`.
- [ ] Roadmap, README, spec "Depois da entrega"; commit `docs(roadmap): contratos do PNCP`.
