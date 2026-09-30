# Licenças, alertas com filtro e vínculos — plano

Spec: `docs/superpowers/specs/2026-09-29-licencas-alertas-e-vinculos-design.md`.
Execução nesta sessão, uma tarefa por commit, TDD.

## Task 1: tipo `licenca_ambiental`
- `domain/act.go`: `ActLicencaAmbiental`, válido.
- `parser/classify.go`: regra do título + marcador; fallback para `outro`
  com "torna público que … licença". Testes em `parser/classify_test`.
- Rótulos: `http/feed.go`, MCP (`tipo` de `buscar_atos`), web
  (`api.ts`, `types.ts`, botão "Licenças ambientais" na busca).

## Task 2: alerta com filtros
- Migration 032: `subscriptions` com `filter_type`, `filter_organ`,
  `filter_theme`, `filter_source` (texto, padrão vazio).
- `domain.Subscription.Filter` (`AlertFilter`), `NewSubscription` aceita
  termo vazio com filtro, valida tipo, órgão e tema; `Subject` lista os
  filtros.
- `ActFilter.GazetteID` (interno) e `filterSQL`; `MatchSubscriptions`
  usa `ActRepository.Search` quando há filtro.
- HTTP `POST /v1/subscriptions` com `filters`; DTO devolve os filtros.
- Web: `subscribe(email, query, filters)`; o formulário de alerta aparece
  com termo ou filtro e diz quais filtros entram.

## Task 3: tema na busca do site
- `searchState.ts`: `theme` na URL, na API, na exportação e no RSS.
- `SearchPage.tsx`: seletor de tema.

## Task 4: padrão de licença a empresa nova
- `domain`: `PatternNewCompanyLicense`, `NewCompanyLicenseDays = 730`,
  `FindNewCompanyLicenses`, finding e catálogo.
- `postgres`: `LicenseActs` (concessões com CNPJ ligado).
- `usecase/list_patterns.go`: relatório novo.

## Task 5: padrão de sócio com nome de agente público
- Migration 033: view materializada `partner_appointment_names`.
- `postgres`: `PartnerAppointments`, `PoliticalAgentNames`, refresh.
- `domain`: `PatternPartnerPublicAgent`, `FindPartnerPublicAgents`.
- Jobs `receita` e `reindex` atualizam a view.

## Task 6: docs e verificação
- README, `parser-findings.md`, roadmap, fontes do MCP.
- `make lint test test-integration`, typecheck, vitest, reindexação
  local e conferência dos números; PR.
