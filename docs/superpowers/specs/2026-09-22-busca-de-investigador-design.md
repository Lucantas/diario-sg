# Busca de investigador (Entrega 1)

Data: 2026-09-22. Parte da Entrega 1 de `docs/roadmap.md` ("Citável e
exportável"): faixa de valor, operadores booleanos e URL permanente para
cada consulta.

## Objetivo

- Filtrar atos por **faixa de valor** em reais, a partir dos valores já
  extraídos em `act_entities`.
- Deixar os **operadores** da busca utilizáveis e documentados: `OU`
  (além de `OR`), `-termo` para excluir e aspas para frase.
- Toda busca tem uma **URL permanente** (termo, filtros e página), que pode
  ser compartilhada e reaberta com o mesmo resultado (salvo atos novos).
- O front ganha os filtros que a API já tinha e o site não mostrava
  (período) e **paginação** (hoje só aparecem os 20 primeiros).

## Fora do escopo

- Exportação (PR seguinte, que reaproveita estes filtros).
- Faixa de valor na página da empresa.
- Busca por campo (`orgao:SEMED`, `tipo:contrato` dentro do texto): os
  filtros já cobrem isso.

## Fatos que orientam o desenho

- `act_entities` tem 60.001 valores na base local (kind `valor`,
  `normalized` em centavos, só dígitos, de 0 a R$ 5,5 bilhões).
- `websearch_to_tsquery` já entende `or`, `-termo` e aspas:
  `limpeza OR coleta -urbana "lixo hospitalar"` vira
  `'limpez' | 'colet' & !'urban' & 'lix' <-> 'hospital'`. `OU` é stopword
  em português e some: `limpeza OU coleta` vira `'limpez' & 'colet'`, o
  contrário do que o usuário quis.
- O ramo literal da busca (`ILIKE`, termos com dígito ou 8+ caracteres)
  compara a consulta inteira; com operadores ele não casa nada, o que é
  inofensivo porque o ramo textual continua valendo.
- `Search` e `CountByMonth` repetem o mesmo `WHERE` de filtros; exportação
  e RSS vão precisar dele também.

## Decisões

1. **Um ato casa a faixa de valor se tiver ao menos um valor dentro dela.**
   Um extrato cita valor global, mensal e unitário; exigir que todos
   estejam na faixa esconderia o contrato. A linguagem da interface diz
   "cita valor entre…".
2. **API recebe reais em formato de máquina** (`min_value=1500.50`, ponto
   decimal, sem milhar). O front aceita o formato brasileiro
   (`1.500,50`) e converte. Um parâmetro de API ambíguo (`1.500` é mil e
   quinhentos ou um e meio?) não serve para URL permanente.
3. **`OU` maiúsculo vira `or`** fora de aspas, no domínio
   (`TranslateOperators`), aplicado na busca, nas estatísticas e no
   casamento de alertas. Minúsculo continua stopword: "ou" é comum no
   texto dos atos e quem escreve operador costuma escrever em caixa alta,
   como no Google.
4. **Filtros num construtor único** (`filterSQL`) no adapter do Postgres,
   usado por `Search` e `CountByMonth` (e depois por exportação e RSS).
5. **URL em português no site** (`/?q=…&tipo=…&orgao=…&de=…&ate=…&valor_min=…&valor_max=…&pagina=…`);
   a API continua com os nomes em inglês.
6. **Cada item traz os valores citados** (`values_cents`, do maior para o
   menor), para quem filtra por valor ver o que casou.

## Desenho

### Domínio

- `ActFilter` ganha `MinCents, MaxCents int64` (0 = sem limite).
  `Normalize`: negativo ou `MinCents > MaxCents` (com os dois definidos)
  é `ErrInvalidFilter`; aplica `TranslateOperators` em `Query`.
- `TranslateOperators(q string) string`: troca a palavra `OU` (só
  maiúscula, isolada) por `or` fora de aspas.
- `ActHit` ganha `ValuesCents []int64`.
- `MatchSubscriptions` passa a consulta da inscrição por
  `TranslateOperators` antes de `SearchInGazette`.

### Dados

Sem migration. `filterSQL(f, next int) (string, []any)` devolve
`AND … ` com tipo, órgão, período e valor, numerando os parâmetros a
partir de `next`. O filtro de valor:

```sql
AND EXISTS (SELECT 1 FROM act_entities v
            WHERE v.act_id = a.id AND v.kind = 'valor'
              AND v.normalized::bigint >= $n AND v.normalized::bigint <= $m)
```

(só as comparações do limite definido). `Search` seleciona
`values_cents` com `array_agg(... ORDER BY ... DESC)`.

### API

- `/v1/acts` e `/v1/stats/acts`: `min_value`, `max_value` (reais,
  `^\d+(\.\d{1,2})?$`); formato inválido é 400.
- Cada item ganha `values_cents` (lista, vazia quando não há).

### Front

- `src/searchState.ts`: `SearchState` (`q`, `type`, `organ`, `from`, `to`,
  `min`, `max`, `page`), `stateFromQuery(search)`, `queryFromState(state)`,
  `parseBRL(texto)` → reais em formato de máquina ou `null`, e
  `apiParams(state)`.
- `SearchPage` sai de `App.tsx` para `SearchPage.tsx`. Ao carregar com
  parâmetros na URL, busca direto; cada busca faz `pushState`; o botão
  voltar (`popstate`) restaura a busca anterior.
- "Mais filtros" (`<details>`): de/até (`type=date`) e valor mínimo/máximo
  (texto, formato brasileiro, com erro visível se inválido). Abre sozinho
  quando algum desses filtros está ativo.
- Paginação: "Anterior", "Página N de M", "Próxima" (20 por página).
- Dica de operadores sob a busca: `"frase exata"`, `OU`, `-excluir`.
- No resultado, "Valores citados: R$ 1.200,00 · R$ 300,00" (até 3, e "+N").

## Testes

- Domínio: `TranslateOperators` (OU isolado, dentro de palavra, minúsculo,
  dentro de aspas); `Normalize` com faixa de valor inválida.
- HTTP: `parseReais` (inteiro, decimal, vírgula, negativo, letras).
- Integração, com uma edição de três contratos (R$ 1.000,00, R$ 50.000,00
  e R$ 200.000,00): `min_value`/`max_value` sozinhos e juntos;
  `values_cents` no item; `limpeza OU coleta` e `limpeza OR coleta` dão o
  mesmo resultado; `-urbana` exclui; `stats` respeita a faixa.
- Front (Vitest): ida e volta `queryFromState`/`stateFromQuery`;
  `parseBRL` com `1.500,50`, `1500`, `1500,5`, vazio e lixo.
- Manual: abrir uma URL com todos os filtros, conferir que o formulário é
  preenchido e o resultado aparece; voltar/avançar do navegador.

## Riscos

- **Valor extraído não é o valor do contrato**: a faixa filtra por
  qualquer valor citado. A interface diz "cita valor"; a regra fica em
  `docs/decisoes-de-codigo.md`.
- **Custo da faixa de valor**: `EXISTS` por ato usa a chave primária
  `(act_id, kind, normalized)`; com filtro de texto é barato, sem texto
  percorre os atos do período. Aceitável no volume atual (150 mil atos).
