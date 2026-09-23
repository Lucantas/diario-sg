# Dump periódico da base (Entrega 1)

Data: 2026-09-23. Parte da Entrega 1 de `docs/roadmap.md` ("Citável e
exportável"): dump completo e periódico, publicado no bucket.

## Objetivo

Uma vez por semana a base inteira (edições, atos com texto completo e
entidades extraídas) é publicada em arquivos abertos, com um manifesto de
tamanhos e hashes, num endereço fixo do site (`/dados/`). Uma página do
site explica os arquivos e mostra a data do último dump.

## Fora do escopo

- Arquivo SQLite ou Parquet prontos. O roadmap pedia "CSV/Parquet e um
  SQLite pronto para o Datasette"; o CSV compactado resolve os dois casos
  sem dependência nova no Go (`sqlite-utils insert … --csv` e
  `pandas.read_csv` leem direto), e o LEIAME mostra os comandos. SQLite e
  Parquet ficam para quando alguém pedir.
- Histórico de dumps: só o último fica publicado. Os PDFs originais já
  estão arquivados e têm hash; o dump é reproduzível a partir deles.
- As inscrições em alertas (e-mails) nunca entram no dump.

## Fatos que orientam o desenho

- Base local: 4.301 edições, 153.750 atos (corpo médio de 1,1 mil
  caracteres; 176 MB de texto), cerca de 70 mil entidades.
- `pkg/gcp.Storage.Put` envia o corpo de um `io.Reader`, então aceita um
  `io.Pipe` e não precisa de arquivo temporário. No Cloud Run o `/tmp` é
  memória.
- O bucket de edições é privado e move tudo para ARCHIVE em 30 dias; não
  serve para arquivos públicos lidos com frequência.
- O Cloud Scheduler tem 3 jobs grátis; hoje há 1 (scraper).
- O `id` do ato muda a cada reindexação; `(gazette_id, position)` é o que
  identifica um ato entre dumps, salvo mudança de segmentação.

## Decisões

1. **CSV RFC 4180 compactado com gzip**, separador vírgula, UTF-8 sem BOM:
   o dump é para programas. A exportação da busca continua no formato do
   Excel em pt-BR, que é para gente.
2. **Bucket público próprio** (`<projeto>-<prefixo>-dumps`), leitura
   anônima só de objeto (`roles/storage.legacyObjectReader` para
   `allUsers`, que não permite listar), sem lifecycle de arquivamento.
3. **O site serve o dump em `/dados/`**, por proxy do nginx para o
   bucket. O endereço não muda se o bucket mudar.
4. **Manifesto por último**: os arquivos de dados sobem primeiro e
   `manifest.json` (data, linhas, bytes e SHA-256 de cada arquivo) por
   último, para quem lê o manifesto encontrar os arquivos que ele
   descreve.
5. **Job semanal** (domingo, 4h de Brasília), com a imagem da API e uma
   conta própria (`dump`): lê o segredo do banco e escreve só no bucket de
   dumps. Nada de escrita no bucket de edições.
6. **O caso de uso não conhece CSV**: ele pede a cada tabela que se
   escreva num `io.Writer` (porta `DumpSource`, implementada no adapter do
   Postgres), compacta, calcula hash e tamanho, e envia (porta
   `ObjectWriter`).

## Desenho

### Portas e caso de uso

```go
type DumpSource interface {
	Snapshot(ctx context.Context) (DumpSnapshot, error)
}

type DumpSnapshot interface {
	Tables() []string
	WriteTable(ctx context.Context, table string, w io.Writer) (rows int, err error)
	Close() error
}

type ObjectWriter interface {
	Put(ctx context.Context, name, contentType string, body io.Reader) error
}
```

`usecase.PublishDump.Execute(ctx, now) (domain.DumpManifest, error)`:
abre um `Snapshot` (no Postgres, uma transação `REPEATABLE READ` só de
leitura, para as três tabelas saírem da mesma foto) e, para cada tabela, `io.Pipe` → gzip → `MultiWriter(hash, contador)`; a
goroutine escreve a tabela, o `Put` lê o pipe. Depois sobe `LEIAME.txt`
(texto fixo embutido no binário) e `manifest.json`. Qualquer erro
interrompe sem publicar o manifesto.

`domain.DumpManifest{GeneratedAt time.Time; Files []DumpFile}` e
`DumpFile{Name string; Rows int; Bytes int64; SHA256 string}`.

### Postgres (`adapters/postgres/dump.go`)

Três tabelas, com colunas fixas (nome de tabela vem de um `switch`, nunca
do usuário):

- `gazettes.csv.gz`: `id, edition_number, published_at, is_extra, source_url, pdf_sha256, indexed_at`
- `acts.csv.gz`: `id, gazette_id, position, type, organ, title, page_start, page_end, body`
- `act_entities.csv.gz`: `act_id, kind, value, normalized`

Ordenadas por data/posição para diffs estáveis entre dumps. Página nula
sai vazia.

### Binário e config

`cmd/dump` (mesmo padrão de `cmd/reindex`), `config.RoleDump` exige
`DATABASE_URL` e `DUMPS_BUCKET`. `make dump` roda local contra o emulador.
`scripts/local-setup.sh` cria o bucket `diario-dumps`.

### Infra

- `google_storage_bucket.dumps` + `google_storage_bucket_iam_member`
  `allUsers` → `roles/storage.legacyObjectReader`.
- Conta `dump` (entra no `for_each` das contas e no `deployer_act_as`),
  com acesso ao segredo do banco e `roles/storage.objectUser` no bucket
  de dumps.
- `google_cloud_run_v2_job.dump` (1 CPU, 1 GiB, 3600 s, `max_retries = 1`)
  e `google_cloud_scheduler_job.dump` (`0 4 * * 0`, America/Sao_Paulo),
  com `run.invoker` para a conta do scheduler.
- Web ganha `DUMPS_URL = https://storage.googleapis.com/<bucket>`;
  `deploy.yml` atualiza a imagem do job.

### Web

- nginx: `location /dados/ { proxy_pass ${DUMPS_URL}/latest/; … }`. O
  `Dockerfile` define um `DUMPS_URL` padrão (emulador) como já faz com
  `API_URL`.
- Vite: proxy de `/dados` para o emulador em desenvolvimento.
- Página `/dados` (React): o que tem em cada arquivo, como abrir (pandas,
  sqlite-utils/Datasette), aviso de LGPD, e a lista lida de
  `/dados/manifest.json` (data, tamanho, linhas, SHA-256, link). Link
  "Dados abertos" no rodapé da busca.

## Testes

- Caso de uso (fakes): sobe os três arquivos, `LEIAME.txt` e o manifesto
  por último; o manifesto traz linhas, bytes e SHA-256 do que foi enviado
  (conferido descompactando); erro numa tabela não publica o manifesto.
- Postgres (integração): `WriteTable` das três tabelas gera CSV com o
  cabeçalho certo e uma linha por registro; tabela desconhecida é erro.
- Integração ponta a ponta com o emulador do GCS fica fora do CI; manual:
  `make dump` e abrir `http://localhost:5173/dados`.
- Infra: `terraform fmt -check` e `validate`.
- Front: `npm run typecheck`, `formatBytes` testado com Vitest.

## Riscos

- **Tamanho**: acts em gzip deve ficar na casa de 40–60 MB; o `Put`
  simples do GCS aceita até 5 TB. Se o upload falhar no meio, o arquivo
  antigo continua no lugar (o objeto só é trocado quando o upload termina).
- **Consistência entre arquivos**: as três tabelas são lidas em consultas
  separadas; sem cuidado, uma edição indexada no meio do dump apareceria
  em `gazettes` e não em `acts`. Por isso o `Snapshot`: as consultas
  rodam numa transação `REPEATABLE READ` só de leitura, que dá a mesma
  foto às três. A transação fica aberta durante os uploads (minutos); no
  Neon isso só segura uma conexão.
- **LGPD**: o dump só tem o que já é público no Diário e na busca. O
  LEIAME repete o aviso do README sobre não criar perfis de pessoas.
