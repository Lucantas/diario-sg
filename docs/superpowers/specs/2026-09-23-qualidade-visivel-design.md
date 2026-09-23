# Qualidade visível (Entrega 1)

Data: 2026-09-23. Parte da Entrega 1 de `docs/roadmap.md` ("Citável e
exportável"): marcar no ato os casos conhecidos do parser e um botão
"reportar erro" que abre uma fila de correção.

## Objetivo

- Todo ato com um defeito de extração **conhecido e detectável com
  precisão** mostra um aviso, com a explicação em texto, na busca, na
  página da empresa, na edição, na exportação e no RSS (pela API).
- Qualquer pessoa pode **reportar um erro** num ato (tipo do problema e
  descrição opcional). O reporte entra numa fila no banco, que quem mantém
  o projeto lista e fecha por linha de comando.

## Fora do escopo

- Corrigir os defeitos (isso é trabalho do parser, caso a caso).
- Painel web da fila, autenticação de mantenedor, e-mail a cada reporte.
- Avisar "tabela quebrada entre páginas" (ver Fatos).

## Fatos que orientam o desenho

Medidos na base local (153.750 atos, 22/09/2026):

- **Portaria sem número**: 3.498 atos com o título igual ao verbo
  (`Nomeia:`, `Exonera:`…). O parser só deixa o verbo como título quando
  não achou o `Port. nº` que fecha a portaria abreviada; a regra é exata.
- **Só o título**: 2.320 atos com o corpo igual ao título (975 licitações,
  452 contratos). O texto do ato ficou no ato vizinho ou não foi extraído.
- **Ato longo demais**: 339 atos ocupam 10 páginas ou mais do PDF (um
  "termo de fomento" de 2021 vai da página 19 à 92). Em geral são anexos
  ou atos que o parser não separou.
- **Tabela quebrada entre páginas** (`docs/fase-1-relatorio.md` §2.3): a
  regra candidata ("`Port. nº` cujo corpo não começa com o verbo") acha
  1.209 atos, mas a amostra tem muitos falsos positivos: verbos fora da
  lista (`Exclui:`, `Declarar vago:`), corrigendas legítimas, ato só com
  título. Um aviso errado mina a confiança nos certos; fica de fora.
- O `id` do ato muda a cada reindexação; um reporte precisa sobreviver a
  ela.

## Decisões

1. **Avisos calculados na leitura, no domínio** (`domain.ActWarnings`),
   não gravados no banco: regra nova ou ajustada vale na hora, sem
   migration nem reindexação. A API devolve códigos
   (`sem_numero`, `so_titulo`, `muitas_paginas`); o texto de cada um fica
   no front, e a exportação CSV leva os códigos numa coluna `avisos`.
2. **Reporte identifica o ato por edição + posição + título** (cópia do
   título no momento do reporte), não pelo `id`. Se a edição for apagada,
   o reporte vai junto (`ON DELETE CASCADE`).
3. **Sem e-mail no reporte**: não há como responder quem reportou sem
   pedir dado pessoal, e a correção aparece no próprio site (LGPD).
4. **Proteção contra abuso** sem cadastro: campo-isca escondido (robô que
   preenche é descartado em silêncio), mensagem de até 2.000 caracteres e
   limite de 5 reportes por minuto por cliente (chave pelo primeiro IP do
   `X-Forwarded-For`, que o cliente pode falsificar) e de 60 por minuto no
   total por instância. É simples de propósito: o volume esperado é
   pequeno e o pior caso é lixo na fila, que o CLI descarta.
5. **Fila no banco + CLI** (`cmd/reports`): `-status aberto` lista (data,
   edição, título, tipo, mensagem e link do PDF na página),
   `-close <id> -as resolvido|descartado` fecha.

## Desenho

### Domínio

```go
const (
	WarningNoNumber  = "sem_numero"
	WarningTitleOnly = "so_titulo"
	WarningManyPages = "muitas_paginas"
	ManyPagesThreshold = 10
)

func ActWarnings(title string, titleOnly bool, pageStart, pageEnd int) []string
func IsTitleOnly(title, body string) bool
func IsPortariaVerbTitle(title string) bool
```

A regex dos verbos de portaria abreviada sai do parser para o domínio
(`domain.PortariaVerbRe`), e o parser passa a usá-la de lá: aviso e
parser não podem divergir.

`ActHit` ganha `TitleOnly bool` (a busca calcula no SQL, sem trazer o
corpo: `btrim(a.body) = btrim(a.title)`).

`domain.ErrorReport{ID, GazetteID string; Position int; ActTitle, Kind, Message, Status string; CreatedAt time.Time}`;
`ReportKinds` = `texto_errado`, `tipo_errado`, `orgao_errado`,
`pagina_errada`, `outro`; `NewErrorReport(...)` valida tipo, posição ≥ 0,
título não vazio (até 500) e mensagem (até 2.000, `outro` exige
mensagem).

### Dados (migration `006_error_reports.sql`)

```sql
CREATE TABLE error_reports (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    gazette_id  uuid        NOT NULL REFERENCES gazettes (id) ON DELETE CASCADE,
    position    int         NOT NULL CHECK (position >= 0),
    act_title   text        NOT NULL,
    kind        text        NOT NULL CHECK (kind IN ('texto_errado', 'tipo_errado', 'orgao_errado', 'pagina_errada', 'outro')),
    message     text        NOT NULL DEFAULT '',
    status      text        NOT NULL DEFAULT 'aberto' CHECK (status IN ('aberto', 'resolvido', 'descartado')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    closed_at   timestamptz
);
CREATE INDEX error_reports_open_idx ON error_reports (created_at) WHERE status = 'aberto';
```

`ports.ErrorReportRepository`: `Create`, `List(ctx, status)`,
`Close(ctx, id, status)`. Edição inexistente (violação de FK ou uuid
inválido) vira `ErrNotFound`.

### API

- Itens da busca, da empresa, da exportação JSON e atos da edição ganham
  `warnings: []` (códigos). CSV ganha a coluna `avisos` (códigos
  separados por ` | `), antes de `texto`.
- `POST /v1/reports` `{"gazette_id","position","act_title","kind","message","website"}`
  → `202 {"status":"recebido"}`. `website` preenchido → `202` sem gravar.
  Tipo inválido ou mensagem longa → 400; edição inexistente → 404;
  limite → 429.

### Front

- Aviso em cada resultado com aviso: faixa discreta com o texto do código
  (e o número de páginas, no caso de `muitas_paginas`).
- "Reportar erro" ao lado de "Citar este ato": abre um formulário curto
  (select do tipo, textarea opcional, campo-isca escondido) e, depois de
  enviado, agradece e explica que a correção aparece no site.

### CLI

`cmd/reports` (`make reports`, `make reports STATUS=resolvido`,
`make close-report ID=… AS=resolvido`).

## Testes

- Domínio: cada regra de aviso (com e sem), limites de 9 e 10 páginas,
  página desconhecida; `NewErrorReport` (tipos, tamanhos, `outro` sem
  mensagem); o parser continua passando com a regex movida.
- HTTP: limitador (5 por cliente, sexto é 429; outro cliente passa;
  janela renova); campo-isca não grava.
- Integração: busca devolve `warnings` para ato só com título e para
  portaria sem número; CSV tem a coluna `avisos`; `POST /v1/reports`
  grava, lista pelo repositório e fecha; edição inexistente dá 404.
- Front (Vitest): `warningText` para os três códigos.

## Riscos

- **Limite por IP falsificável**: aceito pelo volume esperado; o limite
  global por instância segura enxurradas.
- **Aviso em ato certo**: `muitas_paginas` pode marcar um anexo legítimo
  e longo; o texto diz "pode incluir", não "está errado".
