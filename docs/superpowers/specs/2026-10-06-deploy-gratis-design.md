# Deploy grátis: GitHub Actions, Render e Neon

Data: 06/10/2026. Origem: o Diário SG precisa estar no ar como vitrine de
portfólio, atualizado sozinho, e a conta no Google Cloud não pode ser
criada (sem cartão). Todo serviço usado aqui funciona sem cartão de
crédito. O Terraform do GCP fica no repositório, sem uso, para quando
houver conta.

## Objetivo

Site, API e MCP públicos, com as edições novas indexadas todo dia, as
cargas auxiliares no calendário de hoje e os alertas por e-mail chegando.
Vitrine, não serviço com compromisso: a primeira visita depois de um
tempo parado pode levar até um minuto.

## Arquitetura

```
GitHub Actions (cron, repositório público)
 ├─ diario-prefeitura, diario-camara   worker em segundo plano + scraper
 ├─ receita, sancoes, tce, pncp, federal, agentes, sicam
 └─ dump → branch "dados" (um commit só, refeito toda semana)
          │ escrevem em
          ▼
     Neon, plano grátis (Postgres 16, aws-us-east-2, 0,25 CU)
          ▲ lê
 Render, plano grátis
 ├─ api   Dockerfile da API (API + MCP), dorme após 15 min
 └─ web   build estático do Vite; reescreve /api, /.well-known/oauth-*
          e /dados para a api e para o branch dados
Resend (alertas a partir do domínio próprio)
```

| Antes (GCP) | Agora |
| --- | --- |
| Cloud Scheduler + Cloud Run Jobs | workflows agendados do GitHub Actions |
| Pub/Sub entre scraper e worker | POST síncrono no worker, no mesmo job |
| Cloud Storage (PDFs, cache do OCR) | emulador GCS dentro do job; cache do OCR em `actions/cache` |
| Cloud Storage (dumps) | branch `dados`, servido pelo `raw.githubusercontent.com` |
| Cloud Run (API) e nginx (web, com proxy) | Render (serviço Docker e site estático com reescritas) |
| Neon pago criado pelo Terraform | Neon grátis criado no painel |
| DLQs | job falho e o e-mail de falha do GitHub |

Os PDFs não ficam guardados. O bucket servia de passagem entre o scraper e
o worker e de cópia arquivada para dois recursos da API: o PDF da edição
(`/v1/gazettes/{id}/pdf`, o link "cópia arquivada" da citação) e a leitura
de uma página (`ReadPage`). Sem bucket, os dois baixam o PDF oficial pelo
`source_url` (seção 6).

## O que entra

### 1. Publicador HTTP

Pacote `pkg/pushhttp`, usado pelo scraper e pela API, que implementa as
duas portas `EventPublisher`. Escolhido por configuração:
`EVENTS=push-http` e `PUSH_BASE_URL=http://localhost:8081`. Sem
`EVENTS`, continua o Pub/Sub de hoje.

- Cada `Publish(tópico, evento)` faz POST em `PUSH_BASE_URL/events/<tópico>`
  com o envelope de push do Pub/Sub (`message.data` em base64,
  `message.messageId`, `message.attributes`, `subscription`), que o
  `PushHandler` já decodifica, e espera a resposta.
- O mapa de tópicos usa os nomes das rotas do worker: `gazette-fetched`,
  `gazette-indexed` e `fetch-completed`.
- 2xx é sucesso. 5xx, 429 e erro de rede são repetidos até 3 vezes, com
  espera de 1, 4 e 16 s; depois o erro volta para quem publicou. Outro 4xx
  volta como erro na hora.
- O timeout de cada POST é de 10 min: a indexação de uma edição escaneada
  com OCR cabe nisso.

Com o worker publicando `gazette-indexed` para ele mesmo, o handler de
`gazette-fetched` espera o de `gazette-indexed` (alertas) responder antes
de terminar. O servidor HTTP do Go atende os dois ao mesmo tempo.

### 2. Workflows

- `.github/workflows/_pipeline.yml` (reutilizável): faz checkout, compila
  os binários uma vez, sobe o `fake-gcs-server` com o diretório de dados
  no runner, cria o bucket, restaura o cache do OCR (`ocr/`), sobe o worker
  em segundo plano, espera o `/healthz`, roda o scraper e salva o cache com
  chave nova (`ocr-<run_id>`, restaurando pelo prefixo `ocr-`).
- `diario-prefeitura.yml` e `diario-camara.yml` chamam o reutilizável com
  `SOURCE` e o cron.
- Um workflow por carga auxiliar roda o binário direto contra o Neon, como
  o `make` faz hoje.
- Todos aceitam `workflow_dispatch`, têm `concurrency` por workflow e
  `timeout-minutes`.
- O ambiente `producao` do GitHub guarda os segredos.

Calendário, convertido de America/Sao_Paulo para UTC, que é o fuso do cron
do GitHub:

| Workflow | Cron do Terraform (BRT) | Cron do GitHub (UTC) | Timeout |
| --- | --- | --- | --- |
| diario-prefeitura | `0 8,13,19 * * 1-6` | `0 11,16,22 * * 1-6` | 60 min |
| diario-camara | `30 21 * * 1-6` | `30 0 * * 0,2-6` | 60 min |
| sicam | `0 5 * * *` | `0 8 * * *` | 60 min |
| sancoes | `0 7 * * *` | `0 10 * * *` | 30 min |
| tce | `0 5 * * 0` | `0 8 * * 0` | 60 min |
| pncp | `0 6 * * 0` | `0 9 * * 0` | 60 min |
| federal | `0 8 * * 0` | `0 11 * * 0` | 30 min |
| dump | `0 4 * * 0` | `0 7 * * 0` | 30 min |
| agentes | `0 9 10 * *` | `0 12 10 * *` | 30 min |
| receita | `0 6 20 * *` | `0 9 20 * *` | 360 min |

O cron do GitHub pode atrasar em horário de pico; para a vitrine, tudo bem.

Os workflows de hoje (`deploy.yml`, `infra.yml`, `reindex.yml`) passam a
rodar só por `workflow_dispatch`, para pararem de falhar a cada push. O
`ci.yml` não muda.

### 3. Dump no branch `dados`

O `cmd/dump` ganha um destino em diretório (`DUMP_DIR`), com os mesmos
nomes que hoje vão para `latest/` no bucket. O workflow `dump` grava esses
arquivos num commit órfão e faz `push --force` no branch `dados`: o
histórico não cresce. O site reescreve `/dados/*` para
`https://raw.githubusercontent.com/<dono>/diario-sg/dados/*`, então a
página de dados e os links não mudam. O destino no bucket continua para o
GCP.

O workflow também confere o tamanho do banco
(`pg_database_size(current_database())`) e falha com aviso acima de
900 MB, para dar tempo de agir antes do limite de 1 GB.

### 4. Banco em 1 GB e a busca

Migration nova (as aplicadas não são editadas):

1. Função `numeric_terms(text)` imutável, que junta as sequências com
   dígito (`[0-9][0-9./-]*[0-9]`) do texto.
2. Coluna `acts.numeric_terms text GENERATED ALWAYS AS
   (numeric_terms(body)) STORED`.
3. Índice `gin (numeric_terms gin_trgm_ops)`.
4. `DROP INDEX acts_body_trgm_idx`.

Na base local: o índice novo tem 20 MB, a coluna 24 MB e o índice removido
224 MB. O banco passa de 1.103 MB para cerca de 920 MB, e o restore no Neon
chega sem as linhas mortas (16% de `acts`), por volta de 800 MB.

Busca (`matchClause` em `search.go`):

- Termo com dígito: o `ILIKE '%termo%'` roda em `numeric_terms`. Medido na
  base local: `100410/2018` acha 1 ato e `28.636.579/0001-00` acha 5.289,
  os mesmos números de hoje.
- Termo sem dígito: só o full-text (`a.search @@ q`). Hoje um termo de 8
  caracteres ou mais também casava por substring no texto. Essa é a mudança
  de comportamento: deixa de casar trecho de palavra no meio de outra.
  Medido: "Construtora" 433 → 431, "secretaria municipal de saude"
  14.026 → 14.022, "dispensa de licitacao" 946 → 946.
- A ordem pela frase exata (`x.exact`) continua com `ILIKE` em `body`, só
  sobre os atos encontrados.

A decisão vai para `docs/decisoes-de-codigo.md`, com as contagens.

### 5. Render

`render.yaml` na raiz:

- `diario-sg-api`: serviço web Docker com `services/api/Dockerfile`,
  plano grátis, região `ohio` (perto do Neon em aws-us-east-2), `healthCheckPath: /healthz`, variáveis `DATABASE_URL`,
  `PUBLIC_WEB_URL` e as da API marcadas `sync: false` (preenchidas no
  painel). Deploy automático ao mudar `main`.
- `diario-sg-web`: site estático com `npm ci && npm run build` em
  `apps/web`, publicando `dist`. As reescritas fazem o papel do nginx de
  hoje, e o código do site não muda (continua chamando `/api`):
  - `/api/*` → `https://<api>.onrender.com/*`
  - `/.well-known/oauth-*` → a mesma rota na api (conector OAuth do MCP)
  - `/dados/*` → o branch `dados`
  - `/*` → `/index.html`

Limites por IP: com a reescrita, a requisição passa pela borda do Render
duas vezes, e a posição do IP do cliente no `X-Forwarded-For` muda. O
`client.go` passa a ler a posição configurada (`TRUSTED_PROXY_HOPS`), com
teste; o valor certo é conferido no ar, olhando o cabeçalho que chega.
Isso fecha a pendência da Etapa A.

### 6. PDF oficial quando não há bucket

`GetGazettePDF.Open` e `ReadPage.Execute` passam a abrir o PDF por um
helper comum: primeiro o storage; se ele falhar, o PDF oficial pelo
`source_url` da edição, por uma porta nova `ports.SourcePDF`
(`Open(ctx, url) (io.ReadCloser, error)`) com um adaptador HTTP
(`adapters/sourcepdf`, User-Agent do projeto, timeout de 2 min, só aceita
200 com corpo). Sem `GAZETTE_BUCKET`, a API usa um storage vazio
(`adapters/nostore`, que responde `gcp.ErrObjectNotFound`), e o cache do
OCR fica sempre vazio: a leitura de página escaneada roda o OCR na hora.
`GAZETTE_BUCKET` deixa de ser obrigatório para a API; o worker e as cargas
continuam exigindo (no Actions, apontam para o emulador).

A cópia deixa de ser arquivada de verdade: se o site oficial tirar o PDF do
ar, o link para de funcionar. Para a vitrine, tudo bem.

### 7. Carga inicial

Depois da migration aplicada na base local e do `VACUUM FULL`, um
`pg_dump` sem dono nem privilégios é restaurado no Neon. As migrations
ficam registradas no dump, então o `migrate` no Neon não reaplica nada.
O cache do OCR começa vazio: as edições antigas já estão no banco.

### 8. Runbook

`docs/deploy-gratis.md`: criar o projeto no Neon (região aws-us-east-2,
limite de 0,25 CU, suspensão após 5 min), os segredos do ambiente
`producao`, o blueprint no Render, o DNS do domínio para o Render e para o
Resend, a carga inicial e como rodar cada workflow à mão.

## Fora

- Reindexar na nuvem. Quando o parser mudar, a reindexação roda local
  contra o Neon, ou num workflow que baixa os PDFs pelo `source_url`;
  escrever esse workflow fica para quando for preciso.
- Manter a API acordada com ping: gastaria as 100 CU-h do Neon.
- Guardar os PDFs de forma permanente.
- Remover o Terraform do GCP.

## Testes

- `pkg/pushhttp` com `httptest`: envelope e rota por tópico, retry em 5xx,
  429 e erro de rede, sem retry em 2xx, erro na hora em outro 4xx e erro
  depois de esgotar as tentativas.
- Integração: worker e scraper de verdade com `EVENTS=push-http`, contra
  uma fonte de teste servindo um PDF do `testdata`. Quando o scraper
  termina, a edição está indexada e a execução registrada.
- Integração da busca com Postgres real: número parcial, CNPJ formatado,
  texto e o `EXPLAIN` sem *Seq Scan* em `acts` para termo com dígito.
- Limite por IP lendo o `X-Forwarded-For` na posição configurada.
- PDF: storage vazio cai no `source_url`; storage com o objeto não chama a
  fonte; fonte com status diferente de 200 vira erro.
- `actionlint` nos workflows, no `ci.yml`.
- Antes de anunciar: busca, página de empresa, alerta e MCP no ar, e o
  tamanho do banco no Neon conferido.
