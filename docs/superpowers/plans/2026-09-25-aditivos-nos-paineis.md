# Aditivos nos painéis Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Mostrar por fornecedor o valor de aditivos e prorrogações, à parte do contratado.

**Architecture:** O domínio lê o valor do aditivo do começo do texto; o painel junta o aditivo à contratação pelos números e soma por ano de publicação; a API e a página ganham `amended_cents`.

**Tech Stack:** Go, Postgres, React.

**Spec:** `docs/superpowers/specs/2026-09-25-aditivos-nos-paineis-design.md`

## Global Constraints

- Sem comentários no código; conventional commits em português, sem link de sessão.
- `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.
- Nada rodando ao final.

---

### Task 1: Valor do aditivo

**Files:** Modify `domain/panel_value.go` (+ teste).

**Interfaces:** `PanelValueAmended`; `AmendmentValueCents(head string, mainValueCents int64, declaredBP int, contractedCents int64) int64`.

- [ ] Testes com os trechos da spec, implementar, commit `feat(api): valor de aditivo e prorrogação`.

### Task 2: Aditivos no painel

**Files:** Modify `domain/supplier_panel.go` (+ teste), `adapters/postgres/panels.go`.

**Interfaces:** `PanelAct.DeclaredIncreaseBP`; `SupplierRow.AmendedCents`, `PanelTotal.AmendedCents`, `SupplierPanel.AmendedCents`.

- [ ] Testes (ano de publicação, republicação, contratação só com aditivos), implementar, commit `feat(api): aditivos e prorrogações no painel de fornecedores`.

### Task 3: API e página

**Files:** Modify `presentation/http/panels.go` (DTO), integração, `apps/web/src/...PanelsPage.tsx`, `panels.ts`, testes.

- [ ] `amended_cents` na resposta, coluna e totais na página, texto da coluna; commit `feat(web): aditivos e prorrogações nos painéis`.

### Task 4: Conferência e docs

- [ ] Conferir na base local os maiores valores de aditivo contra o texto; Playwright da página; roadmap; commit `docs(roadmap): aditivos nos painéis`.
