# Qualidade visível — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Avisos de extração em cada ato (três regras precisas) e uma fila de reportes de erro com formulário no site e CLI para quem mantém.

**Architecture:** `domain.ActWarnings` calcula os avisos na leitura a partir de título, `TitleOnly` e páginas; a regex dos verbos de portaria sai do parser para o domínio. Reportes vão para `error_reports` (migration 006) por `usecase.ReportError`, com limitador em memória na rota; `cmd/reports` lista e fecha.

**Tech Stack:** Go 1.23, PostgreSQL 16, React 18, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-23-qualidade-visivel-design.md`

## Global Constraints

- Zero comentários novos no código e na migration.
- Códigos de aviso: `sem_numero`, `so_titulo`, `muitas_paginas` (≥ 10 páginas).
- Tipos de reporte: `texto_errado`, `tipo_errado`, `orgao_errado`, `pagina_errada`, `outro`.
- Mensagem até 2.000 caracteres; título até 500; `outro` exige mensagem.
- Limite: 5 reportes por minuto por cliente, 60 por minuto por instância.
- Conventional commits em português, sem link de sessão de agente.
- Antes de concluir: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`; parser: `make reindex` não é necessário (a segmentação não muda), mas os testes do parser têm de passar sem alteração.

---

### Task 1: Avisos no domínio

**Files:** `core/domain/warnings.go` + `_test.go`; `adapters/parser/headers.go` (usa `domain.PortariaVerbRe`).

- [ ] Testes: `ActWarnings("Nomeia:", false, 1, 1) == ["sem_numero"]`; `ActWarnings("Exonera a pedido:", …)` idem; `ActWarnings("PORTARIA Nº 1", true, 1, 1) == ["so_titulo"]`; páginas 3–12 → `["muitas_paginas"]`, 3–11 → nenhum; página 0 → nenhum; combinação ordenada `sem_numero, so_titulo, muitas_paginas`; sem avisos → slice vazio (não nil). `IsTitleOnly("A", " A \n") == true`, `IsTitleOnly("A", "A\nB") == false`.
- [ ] Implementar; parser usa `domain.PortariaVerbRe`; `go test ./...` sem mudar testes do parser.
- [ ] Commit `feat(domain): avisos de extração conhecidos em cada ato`.

### Task 2: Avisos na API

**Files:** `core/domain/act.go` (`ActHit.TitleOnly`), `adapters/postgres/acts.go` e `entities.go` (`btrim(a.body) = btrim(a.title)` em `Search` e `ReportByEntity`; `Export` calcula por `IsTitleOnly`), `presentation/http/dto.go` (`warnings` em `actHitDTO` e `actDTO`), `export.go` (coluna `avisos`), `integration/warnings_test.go`.

- [ ] Integração com edição `"Nomeia:\nFULANO para o cargo.\nPORTARIA Nº 5/2026\n"` + um ato só com título: busca devolve `warnings` certos; `/v1/gazettes/{id}` idem; CSV com cabeçalho contendo `avisos`.
- [ ] Ajustar `TestCSVRecord` (nova coluna antes de `texto`).
- [ ] Commit `feat(api): avisos de extração na busca, na edição e na exportação`.

### Task 3: Fila de reportes

**Files:** `migrations/006_error_reports.sql`, `core/domain/report.go` + `_test.go`, `core/ports/ports.go` (`ErrorReportRepository`), `core/usecase/report_error.go` + `_test.go`, `adapters/postgres/reports.go`, `presentation/http/reports.go` + `ratelimit.go` + testes, `router.go`, `cmd/api/main.go`, `integration/reports_test.go`.

- [ ] Testes do domínio (`NewErrorReport`) e do limitador (`newRateLimiter(5, 60, time.Minute, clock)`: 5 do mesmo cliente passam, sexto não; outro cliente passa; depois de um minuto renova; 61º no total não passa).
- [ ] Integração: `POST /v1/reports` válido → 202 e linha na tabela; campo-isca → 202 sem linha; tipo inválido → 400; edição inexistente → 404; `List(ctx, "aberto")` traz o reporte; `Close` muda status e `closed_at`.
- [ ] Commit `feat(api): fila de reportes de erro nos atos`.

### Task 4: CLI da fila

**Files:** `cmd/reports/main.go`, `Makefile` (`reports`, `close-report`), `config.RoleReports`.

- [ ] `-status` (padrão `aberto`) imprime uma linha por reporte: `id`, data, edição, título, tipo, mensagem e o link `PUBLIC_WEB_URL/api/v1/gazettes/<id>/pdf`; `-close ID -as resolvido|descartado`.
- [ ] Conferir na base local reportando um ato pelo site e listando.
- [ ] Commit `feat(api): comando para listar e fechar reportes`.

### Task 5: Front

**Files:** `apps/web/src/warnings.ts` + teste, `components.tsx` (aviso e "Reportar erro"), `api.ts` (`warnings`, `reportError`), `styles.css`.

- [ ] Teste de `warningText(code, hit)`: textos dos três códigos (com o número de páginas).
- [ ] Formulário de reporte com select, textarea (máx. 2.000), campo-isca escondido (`aria-hidden`, `tabIndex=-1`, `autoComplete="off"`), estados enviando/enviado/erro.
- [ ] `npm test && npm run typecheck`; conferir na base local.
- [ ] Commit `feat(web): avisos de extração e botão para reportar erro`.

### Task 6: Documentação

- [ ] README (campo `warnings`, `POST /v1/reports`, `make reports`), decisões (avisos na leitura, regra da tabela quebrada descartada, reporte sem id e sem e-mail, limitador), roadmap. Commit `docs: qualidade visível`.
