# Anunciado × pago Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Três padrões que cruzam o anunciado no Diário com o pago segundo o TCE-RJ.

**Architecture:** Finders puros em `domain/payment_patterns.go`; a `SupplierPatternRepo` ganha pagamentos por credor, CNPJs citados, cobertura e atos sem CNPJ; `ListPatterns` monta os achados; a carga da Receita inclui os credores.

**Tech Stack:** Go, Postgres.

**Spec:** `docs/superpowers/specs/2026-09-26-anunciado-x-pago-design.md`

## Global Constraints

- Sem comentários; commits em português, sem link de sessão, como Lucas Dantas <lucas.lucantas38@gmail.com>.
- Linguagem: "padrão para verificar".
- `make lint test`, `make test-integration`.

---

### Task 1: Domínio, fonte e caso de uso
- [ ] `FindPaidWithoutPublication`, `FindUnpaidContracts`, `FindPaidAboveAnnounced`, `AttributeByName`, `AmendedBySupplier` (+ testes); catálogo e achados.
- [ ] `SupplierPatternRepo`: `PaidCreditors`, `CitedInDiario`, `PaymentsCoverage`, `PanelActsWithoutCNPJ`; Receita inclui credores.
- [ ] `ListPatterns` com os três; integração.
- [ ] Commit `feat(api): padrões anunciado × pago`.

### Task 2: Base local e docs
- [ ] Recarga da Receita com os credores; conferir um caso de cada; Playwright de `/padroes`.
- [ ] Roadmap, spec; commit `docs(roadmap): anunciado × pago`.
