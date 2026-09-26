# Pessoal do TCE-RJ Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Carregar os agregados de pessoal que o município informa ao TCE-RJ e mostrá-los mês a mês em `/pessoal`, ao lado das nomeações e exonerações do Diário.

**Architecture:** O job `tce` ganha um segundo caso de uso que lê `situacao_funcional` por ano (2024 em diante), arquiva a resposta e troca os meses dos anos lidos em `tce_staff`. Um caso de uso de leitura monta o painel por mês juntando as contagens de atos do Diário.

**Tech Stack:** Go (`net/http`, `encoding/json`), Postgres, React.

**Spec:** `docs/superpowers/specs/2026-09-26-pessoal-do-tce-design.md`

## Global Constraints

- Sem comentários no código; conventional commits em português, sem link de sessão; commits como Lucas Dantas <lucas.lucantas38@gmail.com>.
- Migration nova: `018_tce_staff.sql`.
- Fonte `tce_pessoal`; mesma base do TCE (`TCE_BASE_URL`).
- Nenhum dado de pessoa.
- `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npx vitest run`.
- Nada rodando ao final.

---

### Task 1: Modelo, leitura e carga

**Files:** Create `migrations/018_tce_staff.sql`, `domain/staff.go` (+ teste), `usecase/load_staff.go` (+ teste), `postgres/staff.go`, `integration/staff_test.go`. Modify `adapters/tce/source.go` (+ teste), `ports.go`, `cmd/tce/main.go`, `resetTables`.

**Interfaces (produz):**
- `domain.StaffRow { Month time.Time; Unit, Situation, Group string; Headcount int; RemunerationCents int64 }`
- `domain.FirstStaffYear = 2024`, `domain.SourceStaff = "tce_pessoal"`
- `tce.(*Source).Staff(ctx, year int) ([]domain.StaffRow, []byte, error)`
- `ports.StaffRepository { Ready(ctx) error; ReplaceStaffYear(ctx, year int, rows []domain.StaffRow) error }`
- `usecase.NewLoadStaff(src, repo, runs, raw, now).Execute(ctx, from, to int) (domain.FetchRun, error)`

- [ ] Migration:
  ```sql
  CREATE TABLE tce_staff (month date NOT NULL, unit text NOT NULL, situation text NOT NULL, grp text NOT NULL,
    headcount int NOT NULL, remuneration_cents bigint NOT NULL, PRIMARY KEY (month, unit, situation));
  ```
- [ ] Testes do domínio, do adapter e do caso de uso; integração com duas cargas.
- [ ] Commit `feat(api): carga dos agregados de pessoal do TCE-RJ`.

### Task 2: Painel e página

**Files:** Create `domain/staff_panel.go` (+ teste), `usecase/get_staff_panel.go`, `presentation/http/staff.go`, `apps/web/src/StaffPage.tsx`, `staff.ts` (+ teste). Modify `ports.go` (`StaffReader`), `postgres/staff.go`, `router.go`, `cmd/api/main.go`, `App.tsx`, `SearchPage.tsx`, `api.ts`, integração.

**Interfaces (produz):**
- `domain.StaffPanel { Units []string; Unit string; Months []StaffMonth }`, `StaffMonth { Month time.Time; Groups []StaffGroup; Headcount int; RemunerationCents int64; Appointments, Dismissals int }`
- `GET /v1/panels/staff?unit=`

- [ ] Montagem por grupo, na ordem fixa; contagens do Diário por mês.
- [ ] Página com filtro de unidade e aviso da classificação.
- [ ] Commit `feat: painel de pessoal do TCE-RJ`.

### Task 3: Carga real e docs

- [ ] Carga real local; conferir 3 meses contra o JSON; Playwright de `/pessoal` em 1280 e 390 px.
- [ ] Roadmap, README, spec "Depois da entrega"; commit `docs(roadmap): pessoal do TCE-RJ`.
