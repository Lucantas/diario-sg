# Dump periódico da base — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publicar toda semana `gazettes.csv.gz`, `acts.csv.gz`, `act_entities.csv.gz`, `LEIAME.txt` e `manifest.json` num bucket público servido pelo site em `/dados/`, com uma página que explica os arquivos.

**Architecture:** `usecase.PublishDump` abre um `DumpSnapshot` (transação `REPEATABLE READ` no Postgres), compacta cada tabela por `io.Pipe` + gzip, calcula SHA-256 e tamanho e envia por `ObjectWriter` (`pkg/gcp.Storage`). `cmd/dump` é o composition root; um Cloud Run Job semanal o executa.

**Tech Stack:** Go 1.23 (`compress/gzip`, `encoding/csv`, `crypto/sha256`, `embed`), PostgreSQL 16, Terraform, nginx, React 18, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-23-dump-da-base-design.md`

## Global Constraints

- Zero comentários novos no código.
- Sem migration.
- CSV do dump: RFC 4180, vírgula, UTF-8 sem BOM, gzip.
- Manifesto sempre por último; erro antes dele não publica manifesto.
- Nenhuma tabela de inscrições no dump.
- Conventional commits em português, sem link de sessão de agente.
- Antes de concluir: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`, `terraform fmt -check -recursive infra`, `terraform validate` (dev e prod).

---

### Task 1: Caso de uso `PublishDump`

**Files:**
- Create: `services/api/internal/core/domain/dump.go`
- Modify: `services/api/internal/core/ports/ports.go` (`DumpSource`, `DumpSnapshot`, `ObjectWriter`)
- Create: `services/api/internal/core/usecase/publish_dump.go`, `publish_dump_test.go`, `leiame.txt`

**Interfaces:**
- Produces: `domain.DumpManifest{GeneratedAt time.Time; Files []DumpFile}`, `domain.DumpFile{Name string; Rows int; Bytes int64; SHA256 string}` (JSON `generated_at`, `files`, `name`, `rows`, `bytes`, `sha256`); `usecase.NewPublishDump(ports.DumpSource, ports.ObjectWriter) *PublishDump`; `Execute(ctx, now time.Time) (domain.DumpManifest, error)`; objetos em `latest/<nome>`.

- [ ] **Step 1: Testes que falham** — fakes: `fakeSource` com tabelas `{"a": "x,y\n1,2\n", "b": "z\n3\n"}` (linhas = linhas − 1) e `memObjects` que guarda `nome → bytes` e a ordem. Conferências: ordem `latest/a.csv.gz`, `latest/b.csv.gz`, `latest/LEIAME.txt`, `latest/manifest.json`; descompactar `a.csv.gz` dá o CSV original; manifesto com `Rows 1`, `Bytes == len(gz)`, `SHA256 == sha256(gz)` e `GeneratedAt == now`; `fakeSource` com erro na tabela `b` → erro e nenhum `manifest.json`; snapshot fechado em todos os casos.
- [ ] **Step 2:** `go test ./internal/core/usecase/` → FAIL.
- [ ] **Step 3: Implementação**

```go
//go:embed leiame.txt
var leiame string

type PublishDump struct {
	source  ports.DumpSource
	objects ports.ObjectWriter
}

func NewPublishDump(s ports.DumpSource, o ports.ObjectWriter) *PublishDump {
	return &PublishDump{source: s, objects: o}
}

const dumpPrefix = "latest/"

func (uc *PublishDump) Execute(ctx context.Context, now time.Time) (domain.DumpManifest, error) {
	snap, err := uc.source.Snapshot(ctx)
	if err != nil {
		return domain.DumpManifest{}, err
	}
	defer snap.Close()
	manifest := domain.DumpManifest{GeneratedAt: now}
	for _, table := range snap.Tables() {
		file, err := uc.publishTable(ctx, snap, table)
		if err != nil {
			return domain.DumpManifest{}, fmt.Errorf("tabela %s: %w", table, err)
		}
		manifest.Files = append(manifest.Files, file)
	}
	if err := uc.objects.Put(ctx, dumpPrefix+"LEIAME.txt", "text/plain; charset=utf-8", strings.NewReader(leiame)); err != nil {
		return domain.DumpManifest{}, err
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return domain.DumpManifest{}, err
	}
	return manifest, uc.objects.Put(ctx, dumpPrefix+"manifest.json", "application/json", bytes.NewReader(body))
}

func (uc *PublishDump) publishTable(ctx context.Context, snap ports.DumpSnapshot, table string) (domain.DumpFile, error) {
	name := table + ".csv.gz"
	pr, pw := io.Pipe()
	hash := sha256.New()
	counter := &byteCounter{}
	done := make(chan tableResult, 1)
	go func() {
		gz := gzip.NewWriter(io.MultiWriter(pw, hash, counter))
		rows, err := snap.WriteTable(ctx, table, gz)
		if cerr := gz.Close(); err == nil {
			err = cerr
		}
		pw.CloseWithError(err)
		done <- tableResult{rows: rows, err: err}
	}()
	putErr := uc.objects.Put(ctx, dumpPrefix+name, "application/gzip", pr)
	pr.CloseWithError(putErr)
	res := <-done
	if res.err != nil {
		return domain.DumpFile{}, res.err
	}
	if putErr != nil {
		return domain.DumpFile{}, putErr
	}
	return domain.DumpFile{Name: name, Rows: res.rows, Bytes: counter.n, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}
```

(O canal garante que a goroutine sempre termina: se o `Put` falhar no meio, `pr.CloseWithError` faz a próxima escrita no pipe falhar. O erro da tabela vem antes do erro do upload, porque é a causa.) `leiame.txt` descreve os arquivos, as colunas, a chave estável `(gazette_id, position)`, como abrir com pandas e `sqlite-utils` e o aviso de LGPD.

- [ ] **Step 4:** `go test -race ./internal/core/usecase/` → PASS.
- [ ] **Step 5:** commit `feat(api): caso de uso que publica o dump da base`.

### Task 2: Tabelas no Postgres e `cmd/dump`

**Files:**
- Create: `services/api/internal/adapters/postgres/dump.go`
- Create: `services/api/internal/integration/dump_test.go`
- Create: `services/api/cmd/dump/main.go`
- Modify: `services/api/internal/config/config.go` (`RoleDump`, `DumpsBucket`)
- Modify: `services/api/Dockerfile` (binário `dump`), `Makefile` (`dump`), `scripts/local-setup.sh` (bucket `diario-dumps`), `.env.example` (`DUMPS_BUCKET`)

**Interfaces:**
- Produces: `postgres.NewDumpSource(*sql.DB) *DumpSource`; tabelas `gazettes`, `acts`, `act_entities`.

- [ ] **Step 1: Teste de integração que falha** — indexa `investigatorGazette`, abre `Snapshot`, escreve as três tabelas em buffers: `gazettes` tem cabeçalho `id,edition_number,published_at,is_extra,source_url,pdf_sha256,indexed_at` e 1 linha; `acts` tem `id,gazette_id,position,type,organ,title,page_start,page_end,body` e 3 linhas com `page_start` 1; `act_entities` tem `act_id,kind,value,normalized` e linhas de `valor`; `WriteTable(ctx, "subscriptions", …)` é erro.
- [ ] **Step 2:** `make test-integration` → FAIL.
- [ ] **Step 3:** `Snapshot` abre `BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})`; `WriteTable` escolhe a consulta num `switch` e escreve com `csv.Writer`; `Close` faz `Rollback`. Consultas:

```sql
SELECT id, edition_number, published_at, is_extra, source_url, checksum, indexed_at FROM gazettes ORDER BY published_at, source_url;
SELECT a.id, a.gazette_id, a.position, a.type, a.organ, a.title, a.page_start, a.page_end, a.body
  FROM acts a JOIN gazettes g ON g.id = a.gazette_id ORDER BY g.published_at, g.source_url, a.position;
SELECT e.act_id, e.kind, e.value, e.normalized
  FROM act_entities e JOIN acts a ON a.id = e.act_id JOIN gazettes g ON g.id = a.gazette_id
  ORDER BY g.published_at, g.source_url, a.position, e.kind, e.normalized;
```

`published_at` como `AAAA-MM-DD`, `indexed_at` em RFC 3339 UTC, página nula vazia.
`cmd/dump`: carrega `config.RoleDump`, abre banco, `gcp.NewStorage(cfg.DumpsBucket, …)`, executa e loga o manifesto em JSON; erro → `os.Exit(1)`.
- [ ] **Step 4:** `make lint test test-integration` → PASS; `make setup && make dump` na base local; `curl localhost:4443/diario-dumps/latest/manifest.json`.
- [ ] **Step 5:** commit `feat(api): comando dump com as tabelas do Postgres`.

### Task 3: Infra do dump

**Files:**
- Modify: `infra/stack/main.tf`, `infra/stack/variables.tf` (`dump_schedule`), `infra/stack/outputs.tf` (`dumps_bucket`, `dump_job`)
- Modify: `.github/workflows/deploy.yml`

- [ ] **Step 1:** bucket `dumps` (sem `public_access_prevention`), `allUsers` → `roles/storage.legacyObjectReader`; `"dump"` no `for_each` das contas e em `deployer_act_as`; `database_url` IAM para a conta `dump`; `roles/storage.objectUser` no bucket de dumps; `google_cloud_run_v2_job.dump` (comando `/app/dump`, `DUMPS_BUCKET`, `DATABASE_URL` secreto, 1 CPU, 1 GiB, `3600s`, `max_retries = 1`, `ignore_changes` da imagem); `google_cloud_run_v2_job_iam_member.scheduler_runs_dump`; `google_cloud_scheduler_job.dump` com `var.dump_schedule` (padrão `0 4 * * 0`); web ganha `DUMPS_URL = "https://storage.googleapis.com/${google_storage_bucket.dumps.name}"`.
- [ ] **Step 2:** `deploy.yml`: `gcloud run jobs update $PREFIX-dump --image $REGISTRY/api:$TAG`.
- [ ] **Step 3:** `terraform fmt -check -recursive infra` e `validate` (dev, prod).
- [ ] **Step 4:** commit `feat(infra): bucket público, job e agenda do dump semanal`.

### Task 4: `/dados` no site

**Files:**
- Modify: `apps/web/nginx/default.conf.template`, `apps/web/Dockerfile` (`DUMPS_URL`, `DUMPS_HOST`)
- Modify: `apps/web/vite.config.ts` (proxy `^/dados/`)
- Create: `apps/web/src/DataPage.tsx`, `apps/web/src/format.ts`, `apps/web/src/format.test.ts`
- Modify: `apps/web/src/App.tsx` (rota `/dados`), `SearchPage.tsx` (rodapé), `styles.css`

- [ ] **Step 1: Teste que falha** (`format.test.ts`): `formatBytes(0) === "0 B"`, `formatBytes(1536) === "1,5 KB"`, `formatBytes(52_428_800) === "50,0 MB"`.
- [ ] **Step 2:** `npm test` → FAIL.
- [ ] **Step 3:** `formatBytes`; nginx:

```nginx
  location /dados/ {
    proxy_pass ${DUMPS_URL}/latest/;
    proxy_ssl_server_name on;
    proxy_set_header Host ${DUMPS_HOST};
  }
```

`Dockerfile`: `ENV … DUMPS_URL=http://host.docker.internal:4443/diario-dumps DUMPS_HOST=localhost`; infra passa `DUMPS_HOST = "storage.googleapis.com"`. Vite: `"^/dados/": { target: "http://localhost:4443", rewrite: (p) => p.replace(/^\/dados\//, "/diario-dumps/latest/") }`. `DataPage` busca `/dados/manifest.json`, mostra data, tabela de arquivos (nome com link, linhas, tamanho, SHA-256), descrição das colunas, exemplos (`pandas.read_csv("…acts.csv.gz")`, `sqlite-utils insert diario.db acts acts.csv --csv`) e aviso de LGPD; sem manifesto, avisa que o primeiro dump ainda não foi gerado. Rodapé da busca com link "Dados abertos".
- [ ] **Step 4:** `npm test && npm run typecheck`; abrir `/dados` no Vite local depois de `make dump`.
- [ ] **Step 5:** commit `feat(web): página de dados abertos com o dump da base`.

### Task 5: Documentação

- [ ] README (dump, `/dados/`, `make dump`, bucket de dumps nos outputs), decisões (CSV de programa × CSV de planilha, bucket próprio, manifesto por último, snapshot), roadmap. Commit `docs: dump da base`.
