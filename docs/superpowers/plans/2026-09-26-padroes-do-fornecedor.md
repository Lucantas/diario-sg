# Padrões do fornecedor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cinco padrões novos em `/padroes`, cruzando as contratações do Diário com o cadastro da Receita e as sanções da CGU.

**Architecture:** Finders puros no domínio (`supplier_patterns.go`) sobre as contratações dos painéis e perfis de fornecedor; uma fonte Postgres nova (`SupplierPatternRepo`) lê atos dos painéis, `rf_*` e `cgu_sanctions`; `ListPatterns` monta os achados.

**Tech Stack:** Go, Postgres, React (sem mudança de componente).

**Spec:** `docs/superpowers/specs/2026-09-26-padroes-do-fornecedor-design.md`

## Global Constraints

- Sem comentários; commits em português, sem link de sessão, como Lucas Dantas <lucas.lucantas38@gmail.com>.
- Nenhum achado nomeia sócio pessoa física (ADR 0006).
- Linguagem: "padrão para verificar".
- `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npx vitest run`.

---

### Task 1: Domínio

**Files:** `internal/core/domain/supplier_patterns.go` (+ teste), `pattern.go`.

- [x] `SupplierContracts`, `FindNewCompanyContracts`, `FindUndercapitalizedContracts` (`HasShareCapital`, piso R$ 100 mil), `FindSharedPartners` (grupos juntados), `FindSharedAddresses`, `FindSanctionedContracts` (`ReachesSaoGoncalo`).
- [x] Testes com casos que acionam e que não acionam.
- [x] IDs e textos no catálogo; `*Finding` de cada um.

### Task 2: Fonte, caso de uso e integração

**Files:** `internal/adapters/postgres/supplier_patterns.go`, `ports.go`, `usecase/list_patterns.go` (+ teste), `cmd/api/main.go`, integração.

- [x] `SupplierPatternSource { PanelActs; SupplierProfiles; AllSanctions }`.
- [x] `NewListPatterns(src, suppliers)`; ordem: padrões atuais, depois os cinco novos.
- [x] Integração com um caso de cada.
- [x] Commit `feat(api): padrões do fornecedor com a Receita e a CGU`.

### Task 3: Base local e docs

- [x] Conferir um caso de cada na base local e a página `/padroes` (Playwright, 1280 e 390 px).
- [x] Roadmap (Entrega 3 fechada), spec "Depois da entrega"; commit `docs(roadmap): padrões do fornecedor`.
