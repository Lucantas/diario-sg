# ADR 0001 — Monorepo com arquitetura limpa por serviço

**Status:** aceito

## Contexto
O projeto terá vários componentes (coleta, processamento, API, front) que
evoluem juntos, e pode ganhar serviços em outras linguagens.

## Decisão
- Um único repositório com `apps/` (interfaces de usuário), `services/`
  (backends), `pkg/` (código Go compartilhado), `contracts/` (contratos
  entre serviços, independentes de linguagem) e `infra/`.
- Cada serviço segue a mesma divisão: `core` (domain, ports, usecase),
  `adapters` (mundo externo) e `presentation` (HTTP, eventos, CLI).
  O `core` não importa nenhuma outra camada.
- Cada serviço Go é um módulo próprio; `go.work` une tudo no editor.
- O CI roda só o que mudou (filtro por caminho).

## Consequências
- Mudanças que atravessam serviços saem num único PR.
- Contratos de eventos em JSON Schema permitem serviços em Python ou .NET
  sem depender do código Go.
- Há um pouco mais de cerimônia (interfaces, DTOs) do que num CRUD simples;
  o ganho é testar casos de uso sem banco nem nuvem.
