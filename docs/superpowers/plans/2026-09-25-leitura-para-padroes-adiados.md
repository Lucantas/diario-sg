# Leitura para os padrões adiados Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ler na indexação os incisos de dispensa citados e o acréscimo declarado em aditivos, ler o nome do fornecedor na consulta, e com isso fechar os padrões de aditivos acima do limite, emergencial renovada e fracionamento de obras.

**Architecture:** Funções de texto no domínio; a indexação grava dois fatos novos por ato (migration 013); o `PatternRepo` busca os candidatos pelos fatos (sem regex no SQL) e o domínio aplica as regras.

**Tech Stack:** Go, Postgres, React (página `/padroes` já genérica).

**Spec:** `docs/superpowers/specs/2026-09-25-leitura-para-padroes-adiados-design.md`

## Global Constraints

- Sem comentários no código; migrations aplicadas não se editam.
- Conventional commits em português, sem link de sessão.
- "Padrão para verificar", nunca "irregularidade".
- Antes de terminar: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`, `make reindex FROM=2010-01-01 TO=2026-12-31` com comparação de tipos e órgãos.
- Nada rodando ao final (API, worker, Vite).

---

### Task 1: Incisos de dispensa citados

**Files:** Create `services/api/internal/core/domain/legal_basis.go` (+ `_test.go`); Modify `dispensa_text.go`.

**Interfaces:** `type LegalBasis string`; constantes `Art24I`, `Art24II`, `Art24IV`, `Art75I`, `Art75II`, `Art75VIII`; `func LegalBasisOf(body string) []LegalBasis` (ordenado, sem repetição); `CitesValueDispensa` = contém `Art24II` ou `Art75II`; `CitesWorksDispensa` = contém `Art24I` ou `Art75I`.

- [ ] Testes (casos da spec e os atuais de `CitesValueDispensa`), ver falhar, implementar, passar, commit `feat(api): incisos de dispensa citados no ato`.

### Task 2: Acréscimo declarado em aditivo

**Files:** Create `domain/addendum.go` (+ `_test.go`).

**Interfaces:** `func DeclaredIncreaseBasisPoints(t ActType, title, body string) int`; `func AddendumOrdinal(title, body string) int`; `func MentionsRenovation(body string) bool` (reforma).

- [ ] Testes com os trechos reais da spec, implementar, commit `feat(api): acréscimo declarado em aditivo`.

### Task 3: Gravar os fatos na indexação

**Files:** Create `migrations/013_act_legal_basis.sql`; Modify `domain/act.go` (`LegalBasis []LegalBasis`, `DeclaredIncreaseBP int`), `usecase/parse_acts.go`, `adapters/postgres/gazettes.go` (`insertActs`), teste de integração.

- [ ] Integração: indexar um aditivo e uma dispensa emergencial e ler as colunas; commit `feat(api): indexação grava incisos de dispensa e acréscimo declarado`.

### Task 4: Nome do fornecedor

**Files:** Create `domain/supplier_name.go` (+ `_test.go`).

**Interfaces:** `func SupplierNameOf(body string) string`; `func SupplierNameKey(name string) string`.

- [ ] Testes da spec, implementar, commit `feat(api): nome do fornecedor lido do texto`.

### Task 5: Três regras

**Files:** Create `domain/excessive_addenda.go`, `domain/renewed_emergency.go` (+ testes); Modify `domain/dispensa_limit.go`, `domain/split_dispensa.go`, `domain/pattern.go`, `ports/ports.go`, `adapters/postgres/patterns.go`, `usecase/list_patterns.go` (+ teste), `integration/patterns_test.go`.

**Interfaces:**
- `DispensaCategory` (`DispensaGoods`, `DispensaWorks`); `DispensaLimitCents(published, citesLei14133, category)`; `SplitDispensa.Category`.
- `AddendumAct{ActID, ContractKeys []string, Organ, PublishedAt, Title, Body, IncreaseBP}`; `ExcessiveAddendum{ContractKey, Organ, TotalBP, LimitBP, Acts []AddendumAct}`; `FindExcessiveAddenda([]AddendumAct) []ExcessiveAddendum`.
- `EmergencyAct{ActID, CNPJs, Refs []string, Organ, PublishedAt, Body}`; `RenewedEmergency{Supplier, SupplierLabel, Organ string; Contracts []EmergencyContract}`; `FindRenewedEmergencies([]EmergencyAct) []RenewedEmergency`.
- `PatternSource` ganha `AddendumActs(ctx)` e `EmergencyActs(ctx)`.
- Commits por regra.

### Task 6: Reindex, docs e roadmap

- [ ] `make migrate`, contagem de tipos e órgãos, `make reindex FROM=2010-01-01 TO=2026-12-31`, comparar; medir `/v1/patterns`; conferir `/padroes` no navegador; parar tudo.
- [ ] README (tabela da API), `docs/parser-findings.md`, roadmap (pendência: migration 013 e reindex na nuvem); commit `docs(roadmap): padrões adiados`.
