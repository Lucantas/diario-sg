# Regras para agentes

Este arquivo é a fonte de verdade para qualquer agente (Claude, Codex,
Copilot, Cursor…) que trabalhe neste repositório. Se outra instrução
contradisser algo aqui, vale o que está aqui.

## Comentários no código

Comentário não é proibido, mas é desencorajado. Todo comentário vira uma
segunda coisa para manter ao lado do código, e quando os dois divergem o
comentário passa a mentir.

- Escreva código que se explique: nomes claros, funções pequenas, testes com
  nomes que descrevem o comportamento.
- Só comente quando o código for impossível de entender sem o comentário.
  Se parece precisar de um, tente antes renomear, extrair uma função ou
  escrever um teste.
- O porquê de uma decisão vai na mensagem de commit, na descrição do PR ou
  em `docs/` (`docs/decisoes-de-codigo.md`, `docs/parser-findings.md`,
  `docs/adr/`), não no código.
- Não escreva comentário que repete o código, narra o passo a passo, marca
  seção (`// Arrange`) ou deixa TODO.
- Comentários funcionais ficam como a ferramenta exige: `//go:build`,
  `//go:embed`, `//nolint:…`, shebang, os `## ` dos alvos do `Makefile`
  (usados pelo `make help`).
- Ao mexer num trecho, apague os comentários dele que você não escreveria
  hoje.
- Migrations já aplicadas (`services/api/migrations/`) não são editadas,
  nem para tirar comentário.

## Commits e PRs

- Conventional commits em português: `feat(parser): …`, `fix(scraper): …`,
  `docs(roadmap): …`.
- Não coloque link de sessão de agente (`Claude-Session: https://claude.ai/…`)
  nem outro rastro de ferramenta na mensagem de commit ou na descrição do PR.

## Antes de dizer que terminou

```bash
make lint test          # go vet, gofmt e testes unitários
make test-integration   # Postgres real (precisa de `make up`)
cd apps/web && npm run typecheck
```

Mudou o parser? Rode `make reindex FROM=2010-01-01 TO=AAAA-12-31` na base
local e compare tipos e órgãos antes e depois.

## Onde está o contexto

- `README.md`: arquitetura, como rodar, API.
- `docs/parser-findings.md`: como o Diário Oficial é formatado e por que o
  parser faz o que faz.
- `docs/decisoes-de-codigo.md`: o porquê de escolhas que o código não
  explica sozinho (busca, idempotência, infra).
- `docs/roadmap.md`: entregas, o que está feito e o que falta.
- `docs/adr/`: decisões de arquitetura.
