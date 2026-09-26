# Pagamentos do TCE-RJ Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Carregar os empenhos do TCE-RJ (empenhado, liquidado e pago por credor) e mostrá-los na página da empresa, nos painéis e no MCP.

**Architecture:** Job `cmd/tce` baixa o CSV do ano, guarda só pessoa jurídica em `payments` trocando o ano numa transação, arquiva o bruto filtrado e refaz as ligações por CNPJ citado e ano. Leitura agrega por ano e unidade.

**Tech Stack:** Go, Postgres, React, Terraform.

**Spec:** `docs/superpowers/specs/2026-09-26-pagamentos-do-tce-design.md`

## Global Constraints

- Sem comentários; commits em português, sem link de sessão, como Lucas Dantas <lucas.lucantas38@gmail.com>.
- Migration `016_payments.sql`. Fonte `tce_empenhos`; base `https://dados.tcerj.tc.br/api/v1/` (`TCE_BASE_URL`).
- Nenhuma linha de pessoa física no banco ou no bucket.
- `make lint test`, `make test-integration`, web typecheck e vitest.

---

### Task 1: Modelo, leitura e carga
- [x] Migration, `domain/payment.go` (+ teste), `adapters/tce` (+ teste), `usecase/load_payments.go` (+ teste), `postgres/payments.go`, `cmd/tce`, config, Makefile, integração.
- [x] Commit `feat(api): carga dos empenhos do TCE-RJ`.

### Task 2: Leitura na empresa, painéis e MCP
- [x] `PaymentReader { PaymentsByCNPJ; PaymentsCoverage; PaidByCNPJYear; PaidByYear }`; `CompanyReport.Payments`; painel com `PaidCents`; MCP `pagamentos_tce`.
- [x] Web: seção na empresa, coluna nos painéis.
- [x] Terraform, Dockerfile, deploy.
- [x] Commit `feat: pagamentos do TCE-RJ na empresa, nos painéis e no MCP`.

### Task 3: Carga real e docs
- [x] `make tce FROM=2020 TO=2026`; conferir o total de 2025 contra o CSV; Playwright.
- [x] Roadmap, README, spec; commit `docs(roadmap): pagamentos do TCE-RJ`.
