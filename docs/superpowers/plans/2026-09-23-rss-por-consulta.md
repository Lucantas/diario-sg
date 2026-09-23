# RSS por consulta — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `GET /v1/feeds/acts` com os 50 atos mais recentes de qualquer busca, em RSS 2.0, e o link do feed no site.

**Architecture:** `ActFilter.Recent` troca a ordem do `Search` para cronológica; `usecase.ActFeed` fixa limite e ordem; o handler monta o RSS com `encoding/xml` usando `PublicWebURL` para links absolutos.

**Tech Stack:** Go 1.23 (`encoding/xml`), PostgreSQL 16, React 18, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-23-rss-por-consulta-design.md`

## Global Constraints

- Zero comentários novos no código.
- `guid` = `diario-sg:<gazette_id>:<position>`, `isPermaLink="false"`.
- `Cache-Control: public, max-age=900`.
- Conventional commits em português, sem link de sessão de agente.
- Antes de concluir: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.

---

### Task 1: Ordem cronológica e caso de uso do feed

**Files:** `core/domain/act.go` (`Recent`, `FeedLimit`), `core/usecase/act_feed.go` + `_test.go`, `adapters/postgres/search.go` (`orderSQL`), `acts.go`.

- [ ] Teste: `ActFeed` com filtro `{Limit: 20, Offset: 40}` repassa `{Recent: true, Limit: 50, Offset: 0}` (spy de `Search`); filtro inválido devolve `ErrInvalidFilter`.
- [ ] Teste: `orderSQL(domain.ActFilter{Recent: true})` contém `g.published_at DESC` e não `ts_rank`; sem `Recent` é a ordem atual.
- [ ] Implementar; `Search` e `Export` passam a usar `orderSQL(f)`.
- [ ] `make lint test test-integration`; commit `feat(api): ordem cronológica e caso de uso do feed`.

### Task 2: RSS na API

**Files:** `presentation/http/feed.go` + `feed_test.go`, `router.go` (`API.Feed`, rota), `cmd/api/main.go`, `integration/feed_test.go`.

- [ ] Testes unitários:
  - `siteSearchURL("https://site", f)` com termo, tipo, órgão, período e faixa → `https://site/?q=…&tipo=contrato&orgao=SEMED&de=2024-01-01&ate=2024-12-31&valor_min=1500%2C50` (ordem fixa dos parâmetros);
  - `stripMarks("a ⟦b⟧ c") == "a b c"`;
  - `feedItem("https://site", hit)` com página → link `https://site/api/v1/gazettes/g1/pdf#page=3`, guid `diario-sg:g1:7`, título `Contrato: EXTRATO`, categoria `FMS`, `pubDate` `Fri, 18 Sep 2026 12:00:00 -0300`; sem órgão → sem `category`.
- [ ] Integração: `GET /v1/feeds/acts` → `Content-Type` RSS, XML que `xml.Unmarshal` aceita, 3 itens (posições 0, 1, 2), guids `diario-sg:<id>:<pos>`; `?min_value=40000` → 2; `?organ=TOTAL` → 400.
- [ ] Implementar (structs `rss`, `rssChannel`, `rssItem`, `rssGUID` com tags `xml`; `xml.Header` antes; `Cache-Control`).
- [ ] Commit `feat(api): feed RSS de qualquer busca`.

### Task 3: Link do feed no site

**Files:** `apps/web/src/searchState.ts` (`feedUrl`) + teste, `SearchPage.tsx`.

- [ ] Teste: `feedUrl({...EMPTY_STATE, q: "merenda", organ: "SEMED", page: 4})` começa com `/api/v1/feeds/acts?`, leva `q` e `organ`, sem `limit`/`offset`; valor inválido → `null`.
- [ ] Bloco do alerta ganha "Prefere RSS? Assine o feed desta busca" (link absoluto com `window.location.origin`); aparece também sem termo, quando só há filtros.
- [ ] `npm test && npm run typecheck`; conferir o feed num leitor (ou `xmllint`) na base local.
- [ ] Commit `feat(web): link do feed RSS de cada busca`.

### Task 4: Documentação

- [ ] README (rota do feed), decisões (ordem cronológica, `guid` estável, cache). Commit `docs: RSS por consulta`.
