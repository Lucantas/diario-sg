# Ato localizável — página e órgão de cada ato (Entrega 1a)

Data: 2026-09-22. Parte da Entrega 1 de `docs/roadmap.md` ("Citável e
exportável"). É a base das peças 1b (citar e arquivar) e 1c (busca de
investigador), que dependem de saber **onde** cada ato está no PDF e **de
qual órgão** ele é.

## Objetivo

Cada ato gravado passa a ter:

- `page_start` e `page_end`: páginas do PDF (1-based) onde o ato começa e
  termina;
- `organ`: a sigla da seção do Diário em que o ato foi publicado (`SEMAD`,
  `FMS`, `FUNASG`...), vazia quando o ato vem antes da primeira sigla da
  edição (bloco "ATOS DO PREFEITO").

E existe um comando para reprocessar edições já indexadas com o parser
atual, sem apagar edições nem disparar alertas.

## Fora do escopo

- Expor página e órgão na API, link `pdf#page=N`, botão "citar" e servir o
  PDF arquivado: peça 1b.
- Filtro por órgão, nomes por extenso das siglas: peça 1c.
- Separar a sigla seguida do nome por extenso (`SMTC` + `SECRETARIA
  MUNICIPAL DE TURISMO E CULTURA`), limitação já registrada em
  `docs/fase-1-relatorio.md` §2.3.

## Fatos que orientam o desenho

- O extrator (`adapters/pdf`, `pdftotext` sem `-layout`) emite `\f` a cada
  troca de página; a edição de 18/09/2026 tem 11. Hoje `splitLines`
  (`adapters/parser/clean.go`) troca `\f` por `\n` e a informação se perde.
- `isOrganSection` (`adapters/parser/headers.go`) já reconhece a sigla e hoje
  só a usa para fechar o ato anterior. Nas 7 edições de `testdata/editions`
  reconhece `SEMAD`, `SEMCON`, `FMS`, `FUNASG`, `SEMMATRAN` etc., e um falso
  positivo: `ANEXO` (30/06/2026).
- O `id` do ato só é usado como `key` no React; trocar os atos de uma edição
  não quebra link nenhum.
- A indexação é idempotente pelo checksum (`IndexGazette` devolve cedo se a
  edição já existe), por isso melhorias no parser não chegam à base sem um
  mecanismo de reprocessamento. Até aqui isso foi feito apagando edições e
  republicando eventos à mão.

## Decisões

1. **Página levada por dentro do parser.** Cada linha carrega a página de
   onde veio; as funções do parser passam a operar sobre `[]line` em vez de
   `[]string`. Alternativas descartadas: localizar o título do ato nas páginas
   depois do parse (erra justamente nos casos difíceis: título de portaria
   abreviada vem do fim do ato, cabeçalhos quebrados são reunidos antes,
   títulos se repetem) e inserir uma linha-sentinela a cada `\f` (toda função
   que olha "a próxima linha não vazia" teria de pulá-la; cabeçalho quebrado
   na virada de página viraria bug silencioso).
2. **Sigla, sem nome por extenso.** O órgão é a sigla como aparece no Diário.
   A tabela sigla → nome fica para a 1c, junto com o filtro.
3. **Órgão vazio antes da primeira sigla**, em vez de rotular como gabinete:
   não foi verificado que esse bloco é sempre do gabinete do prefeito.
4. **Reindexação é uma funcionalidade**, não um script: vai se repetir a cada
   melhoria do parser.

## Desenho

### Parser (`services/api/internal/adapters/parser`)

- `type line struct { text string; page int }`. `splitLines` numera as
  páginas a partir de 1, incrementando a cada `\f`; as demais funções
  (`stripPageNoise`, `joinSplitHeaders`, `dropPreamble`, `isOrganSection`,
  `startsAct`...) recebem `[]line`. As regras que olham o texto não mudam.
- `joinSplitHeaders`, ao reunir linhas de um cabeçalho quebrado, fica com a
  página da primeira linha.
- `segment` guarda `pageStart` (página da primeira linha com texto) e
  `pageEnd` (página da última linha com texto) e o órgão corrente.
- `Parse` mantém `organ` corrente: vira a sigla quando `isOrganSection` é
  verdadeiro e vale até a próxima sigla. Siglas em `nonOrganSections`
  (inicialmente `ANEXO`) não mudam o órgão; continuam fechando o ato
  anterior, como hoje, para a segmentação não mudar.
- `domain.Act` ganha `PageStart`, `PageEnd int` e `Organ string`.
- A interface `ports.ActParser` não muda.

### Dados (migration `005_act_location.sql`)

```sql
ALTER TABLE acts
    ADD COLUMN page_start int,
    ADD COLUMN page_end   int,
    ADD COLUMN organ      text NOT NULL DEFAULT '';
CREATE INDEX acts_organ_idx ON acts (organ);
```

Páginas nulas identificam atos ainda não reprocessados. `SaveWithActs` e
`ReplaceActs` gravam as três colunas. O parser nunca devolve página 0: texto
sem `\f` é tudo página 1.

### Reindexação

- `ports.GazetteRepository` ganha:
  - `ListByPeriod(ctx, from, to) ([]domain.Gazette, error)`;
  - `ReplaceActs(ctx, gazetteID, editionNumber string, acts []domain.Act) error`:
    numa transação, apaga os atos da edição (entidades saem em cascata),
    insere os novos com entidades e atualiza `edition_number`. A inserção de
    atos e entidades é a mesma de `SaveWithActs`, extraída numa função comum.
- `usecase.ReindexGazettes.Execute(ctx, from, to) (ReindexResult, error)`:
  lista as edições, e para cada uma baixa o PDF de `storage_path`, extrai,
  parseia, extrai entidades, lê o número e chama `ReplaceActs`. Falha numa
  edição é registrada e não interrompe as demais; o erro final junta todas
  (`errors.Join`), como `FetchEditions` do scraper. Não publica
  `gazette.indexed` (não dispara alertas).
- `ReindexResult{Found, Reindexed, Failed int}`.
- `cmd/reindex` (composition root, mesmo padrão de `cmd/migrate`) com `-from`
  e `-to` (`AAAA-MM-DD`, obrigatórios), loga o resultado em JSON.
  `make reindex FROM=... TO=...`.
- Se o PDF não estiver no bucket ou a extração falhar, a edição conta como
  falha e mantém os atos antigos (a transação só abre depois do parse).

## Testes

- Parser, unitários: ato que atravessa `\f` (`PageStart` < `PageEnd`); ato
  antes da primeira sigla com órgão vazio; troca de sigla no meio da página;
  `ANEXO` não vira órgão; cabeçalho quebrado na virada de página fica com a
  página da primeira linha. Os testes existentes, inclusive os de fixtures
  reais (`fixtures_test.go`, `measure_test.go`), continuam passando sem
  mudar a segmentação: mesma quantidade de atos e mesmos títulos.
- Edições reais (`testdata/editions`, pulado quando ausentes, como
  `measure_test.go`): para todo ato, `1 <= PageStart <= PageEnd <= páginas`,
  e a primeira e a última linha do corpo, quando aparecem literalmente no
  texto, aparecem na página `PageStart` e `PageEnd` respectivamente. A
  conferência visual contra o PDF (abrir `pdf#page=N` de alguns atos) fica na
  validação na base local.
- Integração (`internal/integration`): indexar, reindexar e conferir que a
  contagem de atos se mantém, que página e órgão estão preenchidos, que as
  entidades foram refeitas e que nenhum alerta foi enviado.
- Validação na base local: `make reindex FROM=2020-01-01 TO=2026-12-31`;
  resultado sem falhas; total de atos comparado com o de antes (73.526 em
  22/09/2026, antes da migration); distribuição por órgão; nenhum ato de
  edição com `\f` fica com página nula.

## Riscos

- **Virada de página deixa de inserir linha em branco**: o `\n` que fecha a
  última linha de uma página, logo antes do `\f`, não vira mais uma linha
  vazia espúria. Efeito: um cabeçalho quebrado na virada de página (tanto por
  `joinSplitHeaders` quanto pela continuação por preposição pendurada,
  `headerContinues`) agora é reunido num só ato, onde o parser antigo gerava
  dois. Nas 7 edições reais de `testdata/editions` a saída é idêntica à do
  parser antigo (413 atos); o efeito sobre a base inteira é medido pela
  reindexação completa da Task 6.
- **Reindexação completa leva tempo** (1.740 edições, `pdftotext` em cada):
  é sequencial e retomável por período, porque é idempotente.
- **Contagem de atos pode mudar** na reindexação se o parser atual difere do
  que indexou edições antigas; a diferença é reportada, não escondida.
