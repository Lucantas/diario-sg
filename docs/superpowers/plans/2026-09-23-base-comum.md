# Base comum das fontes (etapa B): plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** entidades e ligações com certeza, registro de coletas por evento e a ferramenta `entidade` do MCP para processo e contrato.

**Architecture:** o domínio define chave e certeza, o Postgres grava as ligações na mesma transação dos atos e a migration 008 preenche a base existente com a mesma regra. O scraper publica `fetch.completed.v1`, e o worker grava `fetch_runs`. O MCP lê das ligações.

**Tech Stack:** Go 1.25, PostgreSQL 16, Pub/Sub (emulador local), Terraform, go-sdk de MCP.

**Spec:** `docs/superpowers/specs/2026-09-23-base-comum-design.md`

## Global Constraints

- Zero comentários novos no código; migrations aplicadas não mudam.
- `act_entities`, a busca, o dump e a página do CNPJ não mudam de comportamento.
- Certeza: `exata` (CNPJ), `forte` (processo; contrato com letra), `fraca` (contrato só com números).
- Fonte do Diário: `diario_prefeitura`; registro `ato`; papel `mencionado`.
- Conventional commits em português, sem link de sessão de agente.
- Antes de concluir: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.

---

### Task 1: ADRs, spec e plano
- [ ] Commit `docs(fontes): ADRs, spec e plano da base comum (etapa B)`.

### Task 2: chave e certeza no domínio
**Files:** `domain/entity_key.go` (+ teste).
**Produces:** `type Certainty string` (`CertaintyExact`, `CertaintyStrong`, `CertaintyWeak`); `EntityKey(kind EntityKind, normalized string) string`; `LinkCertainty(kind EntityKind, key string) Certainty`; `ParseEntityInput(kind EntityKind, s string) (string, error)`; `IsLinkedKind(kind EntityKind) bool` (cnpj, processo, contrato); constantes de fonte, registro e papel.
- [ ] Testes com `001/2017`→`1/2017` fraca, `30/fms/2011`→`30/FMS/2011` forte, processo `06.10981/2025-7`→`061098120257` forte, CNPJ exata; entradas inválidas.
- [ ] Implementar; commit `feat(api): chave e certeza das entidades`.

### Task 3: migration 008 e ligador
**Files:** `migrations/008_entities.sql`; `adapters/postgres/gazettes.go` (`insertActs`, `ReplaceActs`), `adapters/postgres/links.go`; `integration/entity_links_test.go`; truncar `entity_links`, `entities`, `fetch_runs` nos helpers de teste.
- [ ] Teste de integração: indexar `gazetteText` gera as ligações esperadas; reindexar não deixa órfãs; SQL de preenchimento sobre `act_entities` dá o mesmo conjunto que o ligador em Go.
- [ ] Implementar; commit `feat(api): entidades e ligações do Diário`.

### Task 4: leitura por entidade e `entidade` do MCP
**Files:** `domain/entity.go` (`EntityReport`); `ports` (`EntityReader`); `usecase/get_entity.go` (+ teste); `adapters/postgres/links.go` (`ReportByKey`); `presentation/mcp/tools.go`, `dto.go`; `integration/mcp_test.go`.
- [ ] Testes: caso de uso valida e normaliza; integração do MCP com `tipo: processo` e `tipo: contrato` (aviso de certeza fraca) e CNPJ.
- [ ] Implementar; commit `feat(api): entidade do MCP aceita processo e contrato`.

### Task 5: coletas
**Files:** `contracts/events/fetch.completed.v1.json`; `pkg/events/events.go`; scraper (`domain`, `ports`, `usecase/fetch_editions.go` + teste, `adapters/gcp/publisher.go`, `config`, `cmd/scraper`); api (`domain/fetch_run.go`, `ports`, `usecase/record_fetch_run.go` + teste, `adapters/postgres/fetch_runs.go`, `presentation/events/push.go` + teste, `cmd/worker`); `GazetteRepo.Coverage` com a última coleta; `presentation/mcp` (`fontes`); `infra/stack/main.tf`; `scripts/local-setup.sh`; `.env.example`.
- [ ] Testes do scraper (publica coleta com números certos, inclusive com falha; erro ao publicar), do worker e de integração (`fetch_runs` idempotente, `fontes` com a última coleta).
- [ ] Implementar; `terraform fmt` e `validate`; commit `feat: registro de coletas por evento`.

### Task 6: documentação e verificação
- [ ] README (evento novo, variável, `entidade`), roadmap e plano (etapa B entregue), `docs/decisoes-de-codigo.md`.
- [ ] Migration na base local: contagens e tempo; `make setup` e uma coleta local chegando em `fetch_runs`.
- [ ] Verificação completa; commit `docs: base comum no README, no roadmap e no plano`.
