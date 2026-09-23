# Base comum das fontes (etapa B)

Data: 2026-09-23. Etapa B de `docs/plano-fontes-publicas.md`. Decisões nas
ADRs 0004 (entidades e ligações), 0005 (coletas e arquivo bruto) e 0006
(LGPD).

## Objetivo

Preparar o que toda fonte nova vai usar, sem mudar o que já funciona:
- entidades e ligações com grau de certeza;
- registro de cada coleta;
- convenção do arquivo bruto;
- regras de dado pessoal.

Para provar o modelo com uso real, a ferramenta `entidade` do MCP passa a
aceitar processo e contrato, além do CNPJ, lendo das ligações.

## Fora do escopo

- Páginas de processo e contrato no site, e alerta por entidade (Entrega 2).
- Qualquer fonte nova (etapa C em diante).
- Tipos de entidade além de `cnpj`, `processo` e `contrato`.
- Mudar `act_entities`, a busca, o dump ou a página do CNPJ.

## Decisões de escolha (tomadas com o usuário)

1. `act_entities` fica; `entities` e `entity_links` entram por cima (ADR 0004).
2. A etapa B entrega a base e a `entidade` do MCP com processo e contrato.
3. `fetch_runs` é gravado pelo worker a partir do evento
   `fetch.completed.v1` publicado pelo coletor (ADR 0005).

## Fatos que orientam o desenho

- Base local: 4.326 CNPJs, 40.391 processos e 1.783 contratos distintos em
  `act_entities`.
- Processo é normalizado só com dígitos; contrato mantém a barra e a sigla
  quando existe (`30/FMS/2011`, `001/2017`).
- O ato muda de id na reindexação (`ReplaceActs` apaga e reinsere).
- O scraper não acessa o banco; ele já calcula encontrados, gravados,
  pulados e com falha (`FetchResult`).
- O relatório por CNPJ (`ReportByEntity`) já monta atos, contagem por tipo
  e soma de valores; o trecho é recortado em volta do valor encontrado.

## Desenho

### Dados (migration 008)

- `entities (id uuid, kind text CHECK (kind IN ('cnpj','processo','contrato')),
  key text, created_at, UNIQUE (kind, key))`.
- `entity_links (entity_id → entities, source text, record_kind text,
  record_id text, role text, certainty text CHECK (certainty IN
  ('exata','forte','fraca')), evidence text, linked_at,
  PRIMARY KEY (entity_id, source, record_kind, record_id, role))` e índice
  `(source, record_kind, record_id)`.
- `fetch_runs (id uuid PK vindo do evento, source, requested_from date,
  requested_to date, found, stored, skipped, failed int, error text,
  started_at, finished_at timestamptz)` e índice `(source, finished_at DESC)`.
- Preenchimento: a migration cria as entidades e as ligações do Diário a
  partir de `act_entities` com a mesma regra de `domain.EntityKey` e
  `domain.LinkCertainty`, em SQL.

### Domínio

- `domain.EntityKey(kind, normalized) string`: CNPJ e processo iguais;
  contrato em maiúsculas, sem espaços e sem zeros à esquerda no primeiro
  número.
- `domain.LinkCertainty(kind, key) Certainty`: `exata` para CNPJ, `forte`
  para processo e para contrato com letra, `fraca` para contrato só com
  números.
- `domain.ParseEntityInput(kind, s) (key, error)`: o que a pessoa digita
  vira chave. Um CNPJ inválido é `ErrInvalidCNPJ`. Processo com menos de 5
  dígitos, ou contrato fora do formato, é `ErrInvalidInput`.
- `domain.SourceDiarioPrefeitura = "diario_prefeitura"`, `RecordAct = "ato"`,
  `RoleMentioned = "mencionado"`.
- `domain.EntityReport`: tipo, chave, certeza mais fraca entre as ligações,
  total de atos, contagem por tipo, soma dos valores citados nos atos e os
  atos mais recentes (até 100, com trecho em volta da evidência).
- `domain.FetchRun` e `Coverage.LastRun`.

### Ligador

- Em `GazetteRepo.SaveWithActs` e `ReplaceActs`, na mesma transação:
  - apagar as ligações do Diário dos atos que saem;
  - depois de gravar `act_entities`, gravar entidades (`ON CONFLICT DO
    NOTHING`) e ligações dos atos novos, para CNPJ, processo e contrato;
  - a evidência é o `value` extraído.

### Leitura

- `ports.EntityReader.ReportByKey(ctx, kind, key) (domain.EntityReport, error)`,
  implementado em Postgres sobre `entities` + `entity_links` + `acts`.
  Entidade que não existe é `ErrNotFound`.
- `usecase.GetEntity.Execute(ctx, kind, input)`: valida e normaliza a
  entrada e chama o leitor.
- MCP `entidade`:
  - entrada `{tipo: cnpj|processo|contrato (padrão cnpj), numero}`;
  - saída: tipo, chave, certeza, aviso quando `fraca`, total de atos,
    atos por tipo, soma dos valores citados, 20 atos recentes com fontes,
    e cobertura.

### Coletas

- Contrato `contracts/events/fetch.completed.v1.json` e
  `events.FetchCompleted` em `pkg/events`.
- Scraper:
  - `FetchEditions` publica o evento no fim de `ExecuteRange`, com o
    resultado e o erro;
  - o id da coleta é um UUID gerado no início;
  - falha ao publicar entra no erro retornado;
  - nova variável obrigatória `TOPIC_FETCH_COMPLETED`.
- Worker: `POST /events/fetch-completed` → `usecase.RecordFetchRun` →
  `INSERT … ON CONFLICT (id) DO NOTHING`.
- Terraform: módulo `queue_fetch_completed` (publicador: scraper) e a
  variável no job. Local: tópico no `local-setup.sh` e no `.env.example`.
- `GazetteRepo.Coverage` traz a última coleta do Diário; o `fontes` do MCP
  mostra `ultima_coleta_tentada` (fim, encontrados, gravados, falhas e
  erro).

## Testes

- Domínio: `EntityKey`, `LinkCertainty` e `ParseEntityInput`, com casos
  reais da base (`001/2017`, `30/FMS/2011`, `06.10981/2025-7`).
- Caso de uso: `GetEntity` rejeita entrada inválida e repassa a chave
  normalizada; `RecordFetchRun` repassa a coleta.
- Scraper: `ExecuteRange` publica uma coleta com os números certos, também
  quando há falha; erro ao publicar vira erro da execução.
- Worker: evento válido grava; payload inválido é descartado com 2xx.
- Integração (Postgres real):
  - indexar uma edição cria entidades e ligações iguais às que a
    migration criaria a partir de `act_entities`;
  - reindexar troca as ligações sem deixar órfãs;
  - `ReportByKey` para processo e contrato (com certeza `fraca`);
  - `fetch_runs` idempotente;
  - `fontes` e `entidade` pelo cliente MCP.
- Manual na base local: rodar a migration 008, conferir contagens contra
  `act_entities` e medir o tempo da migration e do `entidade`.
