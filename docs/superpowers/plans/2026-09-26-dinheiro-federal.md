# Dinheiro federal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Carregar toda semana as transferências da União e as emendas parlamentares de São Gonçalo e mostrá-las em `/federal` e na página da empresa.

**Architecture:** Um job novo (`cmd/federal`) reaproveita o download de zip da CGU, lê o arquivo de emendas e os arquivos mensais de transferências, filtra São Gonçalo e pessoa jurídica, arquiva as linhas e troca três tabelas. A leitura monta a página e a seção da empresa.

**Tech Stack:** Go (`archive/zip`, `encoding/csv`), Postgres, React, Terraform.

**Spec:** `docs/superpowers/specs/2026-09-26-dinheiro-federal-design.md`

## Global Constraints

- Sem comentários no código; conventional commits em português, sem link de sessão; commits como Lucas Dantas <lucas.lucantas38@gmail.com>.
- Migration nova: `020_federal.sql`.
- Fontes `cgu_emendas` e `cgu_transferencias`; base `CGU_BASE_URL`.
- Nenhum CPF chega ao banco ou ao bucket.
- `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npx vitest run`.
- Nada rodando ao final.

---

### Task 1: Modelo, leitura e carga

**Files:** Create `migrations/020_federal.sql`, `domain/federal.go` (+ teste), `adapters/cgu/files.go` (+ teste), `usecase/load_federal.go` (+ teste), `postgres/federal.go`, `cmd/federal/main.go`, `integration/federal_test.go`. Modify `ports.go`, `config.go` (`RoleFederal`), `Makefile`, `resetTables`.

- [ ] Leitura dos três CSVs com filtro e descarte de CPF; carga com troca e bruto.
- [ ] Commit `feat(api): carga das emendas e transferências federais`.

### Task 2: Página e empresa

**Files:** Create `usecase/get_federal.go`, `presentation/http/federal.go`, `apps/web/src/FederalPage.tsx`, `AmendmentsSection.tsx`, `federal.ts` (+ teste). Modify `entity.go`, `company_sources.go`, DTOs, MCP, `cmd/api/main.go`, `App.tsx`, `SearchPage.tsx`, `CompanyPage.tsx`, `api.ts`.

- [ ] `GET /v1/federal`; `amendment_payments` na empresa; MCP `emendas_pagas_cgu`.
- [ ] Commit `feat: dinheiro federal na página própria e na empresa`.

### Task 3: Nuvem, carga real e docs

- [ ] Terraform (job `federal`, `0 8 * * 0`), Dockerfile, deploy.
- [ ] Carga real; conferir totais; Playwright.
- [ ] Roadmap, README, spec; commit `docs(roadmap): dinheiro federal`.
