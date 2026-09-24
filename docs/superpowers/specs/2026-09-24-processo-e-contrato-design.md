# Páginas de processo e de contrato (Entrega 2)

Data: 2026-09-24. Primeiro subprojeto da Entrega 2 de `docs/roadmap.md`.
Lê de `entities` e `entity_links` (ADR 0004); não traz fonte nova.

## Objetivo

Quem investiga abre `/processo/387-2022` ou `/contrato/30-FMS-2011` e vê,
na ordem em que aconteceram, os atos do Diário que citam aquele número:
edital, homologação, extrato, aditivos, rescisão e os outros atos do
processo. Cada ato mostra a fase, o órgão e o link para a página do PDF.
Os números de processo e de contrato nos resultados da busca viram links
para essas páginas. A ferramenta `entidade` do MCP ganha a mesma fase e o
mesmo agrupamento por órgão.

## Fora do escopo

- Alerta por entidade, painéis e padrões para verificar (próximos
  subprojetos da Entrega 2).
- Mudar a chave de processo ou de contrato (ADR 0004 fica como está).
- Gravar a fase no banco. Fica para quando um filtro ou padrão precisar
  dela em SQL; a função de domínio passa então a ser chamada na indexação.
- Soma de valores nas páginas de processo e contrato. A soma de tudo o
  que é citado conta o mesmo valor várias vezes (commit `c490c88`); cada
  ato continua mostrando os seus valores.
- Página de CNPJ (`/empresa/{cnpj}`), que continua como está.

## Decisões de escolha (tomadas com o usuário)

1. Uma página por número. Quando o número aparece em mais de um órgão,
   os atos são agrupados por órgão, com filtro por órgão e um aviso de
   que podem ser processos diferentes ou uma ata compartilhada.
2. A fase é calculada na leitura, por uma função pura de domínio, sem
   migration nem reindexação.

## Fatos que orientam o desenho

Base local em 24/09/2026:

- 41.077 processos, 1.511 contratos e 4.394 CNPJs em `entities`.
- 82% dos processos têm um ato só. O maior processo tem 189 atos; o
  maior contrato, 196. O maior CNPJ tem 5.296.
- Entre os processos com mais de um ato com órgão identificado, 3.748
  aparecem em um órgão e 3.469 em dois ou mais. O `2808/2022` tem atos
  da SEMAD, SEMTRAN, SEMMATRAN, FAELSG, SEMED e SEMMA.
- A chave do processo é só dígitos (`3872022`); o texto original
  (`PROCESSO ADMINISTRATIVO N.º 387/2022`, `processo SEI nº
  03.03985/2025-0`) fica em `act_entities.value`.
- Os atos ligados a processo são, na maioria, fora do fluxo de
  contratação: 15.433 despachos, 14.324 portarias, autos de infração,
  concessões de licença.
- O tipo do ato não basta para a linha do tempo. `HOMOLOGAÇÃO`, `EXTRATO
  DA ATA DE REGISTRO DE PREÇOS` e `AVISO DE LICITAÇÃO` são `licitacao`;
  `EXTRATO DE DISTRATO DE CONTRATO`, `EXTRATO DE AJUSTE DE CONTAS E
  RECONHECIMENTO DE DÍVIDA` e `EXTRATO DE NOMEAÇÃO DE FISCAL` são
  `contrato`.

## Desenho

### Domínio

- `domain.Phase` e `domain.PhaseOf(t ActType, title string) Phase`. A
  função olha o título sem acento e em maiúsculas, nesta ordem, e fica
  com a primeira regra que casar:

  | Fase | Regra |
  | --- | --- |
  | `rescisao` | `RESCIS` ou `DISTRATO` |
  | `aditivo` | tipo `aditivo`, ou `ADITIV`, `ADITVO`, `APOSTIL` |
  | `ajuste_contas` | `AJUSTE DE CONTAS` ou `RECONHECIMENTO DE DIVIDA` |
  | `fiscal` | `FISCAL` |
  | `homologacao` | `HOMOLOG` ou `ADJUDIC` |
  | `ata_registro_precos` | `REGISTRO DE PRECO` |
  | `dispensa` | tipo `dispensa`, ou `DISPENSA`, `INEXIGIBILIDADE`, `RATIFIC` |
  | `contrato` | tipo `contrato` |
  | `licitacao` | tipo `licitacao`, `edital` ou `ata` |
  | `outro` | o resto |

  `domain.Phases` lista as fases nessa ordem de processo (licitação,
  homologação, ata, dispensa, contrato, fiscal, aditivo, ajuste de
  contas, rescisão, outro), para o resumo.
- `domain.EntityLabel(kind, value string) string`: o número como está
  escrito no ato (`387/2022`, `03.03985/2025-0`, `001/2016`), sem o
  prefixo (`PROCESSO ADMINISTRATIVO N.º`) e sem quebra de linha. Para
  CNPJ, devolve o CNPJ formatado.
- `ParseEntityInput` para contrato aceita `-` no lugar de `/`, para a URL
  do site (`/contrato/30-FMS-2011`). Processo já ignora o que não é
  dígito.
- `domain.ActHit` ganha `Phase` (preenchida só no relatório de entidade)
  e `Mentions []EntityMention{Kind, Key, Label}` com os processos e
  contratos citados no ato.
- `domain.EntityReport` ganha:
  - `Label`: o rótulo mais frequente entre as ligações;
  - `Organs []OrganCount{Organ, Acts}` sobre todos os atos ligados, com
    `""` para os atos sem órgão;
  - `CountByPhase map[Phase]int` sobre todos os atos ligados;
  - `Related []RelatedEntity{Kind, Key, Label, Acts}`: contratos e
    CNPJs citados nos atos de um processo; processos e CNPJs citados nos
    atos de um contrato. Até 20 de cada tipo, pelos que têm mais atos.

### Leitura

- `LinkRepo.ReportByKey`:
  - limite de atos por tipo de entidade: 300 para processo e contrato
    (cabe o maior hoje), 100 para CNPJ, como agora;
  - órgãos e tipos contados sobre todas as ligações, não só sobre os
    atos devolvidos;
  - `Related` numa consulta por tipo relacionado, parecida com a de
    `linkedProcesses`; `ByProcess` do CNPJ não muda;
  - o rótulo sai de `act_entities.value` dos atos ligados.
- A busca (`ActRepo.Search`) e `linkedActs` trazem os processos e
  contratos de cada ato, numa subconsulta como a dos CNPJs.
- O leitor devolve também `TypeTitleCounts []TypeTitleCount{Type, Title,
  Acts}`, a contagem de todos os atos ligados por tipo e título.
  `GetEntity` preenche `Phase` de cada ato e soma `CountByPhase` a partir
  dessa contagem. Assim a regra da fase fica só no domínio.

### API HTTP

- `GET /v1/entities/{kind}/{key}` para `processo` e `contrato`, ao lado
  da rota de CNPJ, que não muda. Tipo desconhecido é 404; número
  inválido é 400.
- Resposta:

  ```json
  {
    "kind": "processo", "key": "3872022", "label": "387/2022",
    "certainty": "forte", "diarios": 1, "total_acts": 5,
    "count_by_phase": {"licitacao": 2, "outro": 3},
    "organs": [{"organ": "FMS", "organ_name": "…", "acts": 5}],
    "related": [{"kind": "contrato", "key": "…", "label": "…", "acts": 2}],
    "acts": [ /* actHitDTO com "phase" */ ]
  }
  ```

- `actHitDTO` ganha `phase` (omitido fora do relatório de entidade) e
  `mentions: [{kind, key, label, slug}]`, onde `slug` é o rótulo com `/`
  trocado por `-`, pronto para a URL do site.

### Site

- Rotas `/processo/{slug}` e `/contrato/{slug}` em `App.tsx`, com um
  componente `EntityPage`.
- Cabeçalho com o rótulo e o tipo. Avisos, quando for o caso:
  - número em mais de um órgão: "Este número aparece em N órgãos. Podem
    ser processos diferentes com o mesmo número, ou uma ata de registro
    de preços usada por vários órgãos. Confira o órgão em cada ato.";
  - contrato sem sigla (certeza `fraca`) e número nos dois Diários: os
    mesmos textos da `entidade` do MCP.
- Resumo por fase, na ordem de `Phases`.
- Filtro por órgão (botões com a contagem, "Todos" por padrão), guardado
  na URL (`?orgao=SEMAD`). Com mais de um órgão, os atos aparecem em
  grupos por órgão, do que tem mais atos para o que tem menos, e "sem
  órgão" por último; dentro de cada grupo, do mais antigo para o mais
  recente.
- Cada ato usa o `Result` da busca, com a fase ao lado do tipo.
- Seção "Citados junto" com os `related`, cada um com link para a
  própria página (`/processo/…`, `/contrato/…`, `/empresa/…`).
- Quando há mais atos que o limite, a página diz "mostrando os 300 mais
  recentes de N".
- No `Result`, os números em `mentions` aparecem como links, ao lado de
  "Empresas citadas".
- Número sem nenhum ato: "Nenhum ato indexado cita este número."

### MCP

- `entidade` ganha `rotulo`, `atos_por_fase`, `orgaos` e
  `citados_junto`, e cada item de `atos_recentes` ganha `fase`.
- O aviso de vários órgãos entra em `aviso`, junto dos que já existem.
  Os textos dos avisos passam para o domínio (`domain.EntityWarnings`),
  para o site e o MCP dizerem a mesma coisa.

## Erros

- Número inválido: 400 com a mensagem de `ErrInvalidInput`.
- Número válido sem nenhuma ligação: 200 com zero atos. "Não aparece no
  Diário" é uma resposta, como já é no MCP.
- Falha de banco: 500, registrada no log, como nas outras rotas.

## Testes

- Domínio:
  - `PhaseOf` com os títulos reais da tabela acima e com títulos
    parecidos que não devem casar (`PORTARIA` com `FISCAL` no corpo,
    não no título, fica `outro`);
  - `EntityLabel` com os valores de `act_entities` citados em "Fatos";
  - `ParseEntityInput` com `30-FMS-2011` e `387-2022`;
  - `EntityWarnings` para vários órgãos, certeza fraca e dois Diários.
- Caso de uso: `GetEntity` preenche a fase de cada ato e soma
  `CountByPhase`.
- Integração (Postgres real):
  - processo com atos em dois órgãos e em um ato sem órgão: `Organs`,
    `CountByPhase` e `Related` certos;
  - processo com mais de 100 atos devolve todos (até 300);
  - `mentions` na busca e no relatório;
  - `GET /v1/entities/processo/387-2022` e `/contrato/30-FMS-2011`;
    tipo desconhecido dá 404;
  - `entidade` pelo cliente MCP com `atos_por_fase` e `orgaos`.
- Site (Vitest): agrupar e ordenar atos por órgão, rótulo para slug e
  slug para rota.
- Manual na base local: abrir `/processo/2808-2022`, um processo de um
  ato só e um contrato com aditivos e distrato; conferir as fases contra
  os títulos.
