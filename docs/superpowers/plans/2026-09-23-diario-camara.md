# Diário Oficial da Câmara (etapa C1): plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** coletar e indexar o Diário da Câmara nas mesmas tabelas do da Prefeitura, com busca, MCP e site cientes da fonte.

**Architecture:** `gazettes.source` distingue as fontes. O parser recebe ruído e número de edição pela fonte. O coletor escolhe o adaptador pela variável `SOURCE`, e o evento `gazette.fetched.v1` leva a fonte até o worker.

**Tech Stack:** Go 1.25, PostgreSQL 16, Pub/Sub, Terraform, React + Vite.

**Spec:** `docs/superpowers/specs/2026-09-23-diario-camara-design.md`

## Global Constraints

- Zero comentários novos no código; migrations aplicadas não mudam.
- Fontes: `diario_prefeitura` (padrão em todo lugar onde a fonte falta) e `diario_camara`.
- Caminho no bucket da Câmara: `raw/diario_camara/AAAA/MM/DD/AAAA-MM-DD.pdf`.
- Coletor da Câmara: só URL por data, 2 s entre pedidos, sem contornar o WAF.
- Conventional commits em português, sem link de sessão de agente.
- Antes de concluir: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.

---

### Task 1: ADR, spec e plano
- [ ] Commit `docs(camara): ADR, spec e plano do Diário da Câmara (etapa C1)`.

### Task 2: parser por fonte
**Files:** `adapters/parser/{source.go,clean.go,edition.go,classify.go,regex.go}` (+ `camara_test.go`); `ports` (`ActParsers`); `usecase/{index_gazette.go,reindex_gazettes.go}` (+ testes); `cmd/{worker,reindex}`.
**Produces:** `parser.ForSource(source string) Regex`; `parser.Set{}` com `For(source string) ports.ActParser`; `IndexGazetteInput.Source`; `Gazette.Source`.
- [ ] Testes do parser da Câmara (cabeçalho, rodapé, edição nas três grafias, termo de homologação) e da escolha do parser no caso de uso.
- [ ] Implementar; commit `feat(parser): cabeçalho e número de edição da Câmara`.

### Task 3: fonte na edição, na busca e nas ligações
**Files:** `migrations/009_gazette_source.sql`; `domain/{source.go,gazette.go,act.go,coverage.go}`; `postgres/{gazettes.go,acts.go,filter.go,links.go,dump.go}`; `presentation/events/push.go`; `pkg/events`; `contracts/events/*.json`; `presentation/http/{router.go,dto.go,export.go,feed.go}`; `adapters/email/email.go`; integração.
- [ ] Testes: filtro por fonte no SQL; evento sem fonte é Prefeitura; integração com edição da Câmara (fonte gravada, ligações, busca com `source`).
- [ ] Implementar; commit `feat(api): fonte da edição na busca, nas ligações e nos alertas`.

### Task 4: MCP ciente da fonte
**Files:** `presentation/mcp/{tools.go,dto.go,citation.go}` (+ testes); `usecase/coverage.go`; `integration/mcp_test.go`.
- [ ] Testes: citação da Câmara; `fontes` com dois diários; `buscar_atos` com `diario`.
- [ ] Implementar; commit `feat(mcp): Diário da Câmara nas ferramentas`.

### Task 5: coletor da Câmara
**Files:** scraper `adapters/source/cmsg/source.go` (+ teste), `domain/edition.go`, `domain/fetch_run.go`, `usecase/fetch_editions.go`, `adapters/gcp/publisher.go`, `config`, `cmd/scraper`; `infra/stack/main.tf`, `variables.tf`; `.github/workflows/deploy.yml`; `Makefile`; `.env.example`.
- [ ] Testes do adaptador, do caminho no bucket, da fonte na edição e na coleta, e da config.
- [ ] Implementar; `terraform fmt` e `validate`; commit `feat(scraper): coletor do Diário da Câmara`.

### Task 6: site
**Files:** `apps/web/src/{types.ts,api.ts,searchState.ts,citation.ts,SearchPage.tsx,components.tsx,DataPage.tsx}` (+ testes).
- [ ] Testes de citação por fonte e do estado da busca com `source`.
- [ ] Implementar; commit `feat(web): Diário da Câmara na busca`.

### Task 7: documentação e verificação
- [ ] README, roadmap, plano de fontes, `docs/parser-findings.md` (seção da Câmara).
- [ ] Backfill local da Câmara desde 2020-10-04; contagem por tipo e amostra.
- [ ] Verificação completa; commit `docs(camara): Diário da Câmara no README, no roadmap e no plano`.
