# Sanções da CGU Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Carregar todo dia as sanções do CEIS e do CNEP aplicadas aos CNPJs citados e mostrá-las na página da empresa e no MCP.

**Architecture:** Um job do módulo da API (`cmd/sancoes`) descobre a data do último arquivo de cada cadastro, baixa o zip, filtra as linhas de pessoa jurídica pelo CNPJ básico dos CNPJs citados, arquiva as linhas filtradas e faz *upsert* em `cgu_sanctions` numa transação, refazendo as ligações. A leitura junta as sanções pelo CNPJ básico.

**Tech Stack:** Go (`archive/zip`, `encoding/csv`, `net/http`), Postgres, React, Terraform.

**Spec:** `docs/superpowers/specs/2026-09-26-sancoes-da-cgu-design.md`

## Global Constraints

- Sem comentários no código; conventional commits em português, sem link de sessão; commits como Lucas Dantas <lucas.lucantas38@gmail.com>.
- Migration nova: `015_cgu_sanctions.sql`.
- Fonte `cgu_sancoes`; base `https://portaldatransparencia.gov.br/download-de-dados/` (`CGU_BASE_URL`).
- Nenhuma linha de pessoa física chega ao banco ou ao bucket.
- `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npx vitest run`.
- Nada rodando ao final.

---

### Task 1: Modelo e migration

**Files:** Create `services/api/migrations/015_cgu_sanctions.sql`, `internal/core/domain/sanction.go`, `sanction_test.go`.

**Interfaces (produz):**
- `domain.SourceSanctions = "cgu_sancoes"`, `domain.RecordSanction = "sancao"`
- `type Sanction struct { Register, Code, CNPJ, Name, Category string; StartsAt, EndsAt, PublishedAt *time.Time; Process, Organ, OrganUF, Sphere, Scope, LegalBasis string; FineCents *int64; FirstSeen, LastSeen time.Time }`
- `ParseSanctionRow(header, row []string) (Sanction, bool, error)`: `bool` falso para pessoa física ou sem tipo
- `(Sanction).State(listedOn, today time.Time) SanctionState` com `SanctionActive` ("em_vigor"), `SanctionEnded` ("encerrada"), `SanctionDelisted` ("fora_do_cadastro")

- [ ] Testes: linha do CEIS e do CNEP (cabeçalhos reais), datas `dd/mm/aaaa` e vazias, multa `377157,39` → 37715739, linha `F` descartada, cabeçalho sem coluna obrigatória dá erro; estados.
- [ ] Migration:
  ```sql
  CREATE TABLE cgu_sanctions (register text NOT NULL CHECK (register IN ('CEIS', 'CNEP')), code text NOT NULL, cnpj text NOT NULL, cnpj_base text NOT NULL, name text NOT NULL, category text NOT NULL, starts_at date, ends_at date, published_at date, process text NOT NULL, organ text NOT NULL, organ_uf text NOT NULL, sphere text NOT NULL, scope text NOT NULL, legal_basis text NOT NULL, fine_cents bigint, first_seen date NOT NULL, last_seen date NOT NULL, PRIMARY KEY (register, code));
  CREATE INDEX cgu_sanctions_base_idx ON cgu_sanctions (cnpj_base);
  ```
- [ ] Commit `feat(api): modelo das sanções da CGU`.

### Task 2: Leitura dos arquivos da CGU

**Files:** Create `internal/adapters/cgu/source.go`, `source_test.go`.

**Interfaces (produz):** `cgu.New(baseURL string, client *http.Client) *Source`; `LatestDay(ctx, register string) (time.Time, error)`; `Rows(ctx, register string, day time.Time, each func(header, row []string) error) (sha256 string, err error)` (zip inteiro em memória, SHA-256 do zip, latin-1).

- [ ] Testes contra `httptest`: página com `arquivos.push`, redirecionamento para o zip, CSV latin-1 com cabeçalho; 403 vira erro com o cadastro e a data.
- [ ] Commit `feat(api): leitura dos cadastros de sanções da CGU`.

### Task 3: Carga, job e integração

**Files:** Create `internal/core/usecase/load_sanctions.go` (+ teste), `internal/adapters/postgres/sanctions.go`, `cmd/sancoes/main.go`, `internal/integration/sanctions_test.go`. Modify `ports.go`, `config.go` (`RoleSanctions`, `CGUBaseURL`), `Makefile`, `resetTables`.

**Interfaces (produz):**
- `ports.SanctionSource` (métodos do `*cgu.Source`)
- `ports.SanctionRepository { Ready(ctx) error; CitedCNPJs(ctx) ([]string, error); Save(ctx, load domain.SanctionLoad) error }`
- `ports.SanctionReader { SanctionsByCNPJ(ctx, cnpj string) ([]domain.Sanction, error); SanctionsListedOn(ctx) (map[string]time.Time, error) }`
- `domain.SanctionLoad { Days map[string]time.Time; Sanctions []Sanction }`
- `usecase.NewLoadSanctions(src, repo, runs, raw, now).Execute(ctx) (domain.FetchRun, error)`

- [ ] Teste do caso de uso com fakes: básico, pessoa física fora, bruto e manifesto, falha no CNEP não grava.
- [ ] `Save`: upsert por `(register, code)` mantendo `first_seen`, `last_seen` = dia do arquivo; apaga e refaz as ligações da fonte (`exata` no mesmo CNPJ, `forte` na mesma raiz; evidência = categoria e órgão).
- [ ] `SanctionsListedOn`: `max(last_seen)` por cadastro.
- [ ] Integração: duas cargas, a sanção ausente na segunda fica com o `last_seen` antigo.
- [ ] Commit `feat(api): carga diária das sanções da CGU`.

### Task 4: API, MCP, página e infra

**Files:** Modify `domain/entity.go` (`Sanctions []Sanction`, `SanctionsListedOn map[string]time.Time` em `CompanyReport` e `EntityReport`), `usecase/get_entity.go`, `usecase/queries.go`, `presentation/http`, `presentation/mcp`, `cmd/api/main.go`, integração; `apps/web/src/api.ts`, `sanctions.ts` (+ teste), `SanctionsSection.tsx`, `CompanyPage.tsx`, `styles.css`; Terraform, Dockerfile, deploy.

- [ ] JSON `sanctions` (com `state`) e `sanctions_listed_on`; MCP `sancoes_cgu` e `sancoes_cgu_consultadas_em`.
- [ ] Seção na página, com estado e aviso de outro estabelecimento.
- [ ] Terraform: job `sancoes`, agendamento diário 07:00, conta de serviço; Dockerfile e deploy.
- [ ] Commit `feat: sanções da CGU na página da empresa e no MCP`.

### Task 5: Carga real e docs

- [ ] Carga real local; conferir 10 sanções no Portal da Transparência; Playwright da página de uma empresa sancionada em 1280 e 390 px.
- [ ] Roadmap, README, spec "Depois da entrega"; commit `docs(roadmap): sanções da CGU`.
