# Deploy grátis Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Pôr o Diário SG no ar sem cartão: pipeline e cargas no GitHub Actions, API e site no Render, banco no Neon grátis.

**Architecture:** O scraper e o worker trocam eventos por POST síncrono (`pkg/pushhttp`) no mesmo job do Actions, com o emulador GCS de passagem. A API no Render abre o PDF oficial pelo `source_url` quando não há bucket. O banco cabe em 1 GB trocando o índice trigram do texto inteiro por um só dos trechos numéricos.

**Tech Stack:** Go 1.25 (workspace `pkg`, `services/api`, `services/scraper`), Postgres 16 com `pg_trgm` e `unaccent`, GitHub Actions, Render (blueprint `render.yaml`), Neon.

**Spec:** `docs/superpowers/specs/2026-10-06-deploy-gratis-design.md`

## Global Constraints

- Comentários: só os que a ferramenta exige (`//go:build`, `//go:embed`); nada de comentário explicativo (AGENTS.md).
- Migrations aplicadas não são editadas; a nova é `services/api/migrations/034_numeric_terms.sql`.
- Commits em conventional commits em português, sem link de sessão de agente.
- Sem `EVENTS` (ou `EVENTS=pubsub`), o comportamento de hoje não muda: Pub/Sub, bucket obrigatório para o worker e cargas.
- Antes de cada commit: `make lint test`; tarefas com SQL também `make test-integration` (precisa de `make up`); tarefas com web `cd apps/web && npm run typecheck`.
- Crons do Actions em UTC, convertidos de America/Sao_Paulo (tabela da spec).
- Banco alvo: Neon grátis, 1 GB por projeto; o workflow de dump falha acima de 900 MB.

## Review Focus

- `EVENTS` com valor desconhecido (ex.: `pushhttp`): o serviço deve recusar subir com erro claro, não cair no Pub/Sub em silêncio. Teste na Tarefa 3.
- Worker fora do ar quando o scraper publica: o publicador deve esgotar as tentativas e devolver erro, sem travar o job até as 6 h. Teste na Tarefa 2 (servidor que fecha a conexão) e `timeout-minutes` na Tarefa 9.
- Termo de busca com `%` ou `_` no meio de um número (`12%34`): o `ILIKE` em `numeric_terms` deve tratar como literal, como hoje. Teste na Tarefa 5.
- `X-Forwarded-For` com menos entradas que `TRUSTED_PROXY_HOPS`: deve cair no endereço da conexão, não dar pânico de índice. Teste na Tarefa 6.
- `source_url` vazio ou que responde HTML (página de erro com status 200): o PDF da edição deve dar erro, não entregar HTML como PDF. Teste na Tarefa 7.

---

### Task 1: Porta de publicação genérica

Os adaptadores de eventos dependem hoje do tipo concreto `*gcp.Publisher`. Passam a depender de uma interface, para aceitar o publicador HTTP da Tarefa 2.

**Files:**
- Modify: `pkg/gcp/pubsub.go`
- Modify: `services/api/internal/adapters/pubsub/publisher.go`
- Modify: `services/scraper/internal/adapters/gcp/publisher.go`
- Test: `pkg/gcp/pubsub_test.go` (novo)

**Interfaces:**
- Produces: `gcp.MessagePublisher interface { Publish(ctx context.Context, topic string, data []byte, attrs map[string]string) (string, error) }`; `pubsub.NewEventPublisher(c gcp.MessagePublisher, topicIndexed string)`; `gcp.NewEventPublisher(c gcpclient.MessagePublisher, topic, runsTopic string)` (scraper).

- [ ] **Step 1: Write the failing test**

`pkg/gcp/pubsub_test.go`:

```go
package gcp

import "testing"

func TestPublisherImplementsMessagePublisher(t *testing.T) {
	var _ MessagePublisher = NewPublisher("p", "localhost:8085", nil)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd pkg && go test ./gcp/ -run TestPublisherImplementsMessagePublisher`
Expected: FAIL, `undefined: MessagePublisher`.

- [ ] **Step 3: Implement**

Em `pkg/gcp/pubsub.go`, acima de `type Publisher struct`:

```go
type MessagePublisher interface {
	Publish(ctx context.Context, topic string, data []byte, attrs map[string]string) (string, error)
}
```

Em `services/api/internal/adapters/pubsub/publisher.go`: o campo `client *gcp.Publisher` vira `client gcp.MessagePublisher` e `NewEventPublisher(c *gcp.Publisher, …)` vira `NewEventPublisher(c gcp.MessagePublisher, …)`.

Em `services/scraper/internal/adapters/gcp/publisher.go`: o campo `client *gcpclient.Publisher` vira `client gcpclient.MessagePublisher` e o mesmo no `NewEventPublisher`.

- [ ] **Step 4: Run tests**

Run: `make lint test`
Expected: PASS (os `main.go` continuam passando `*gcp.Publisher`, que satisfaz a interface).

- [ ] **Step 5: Commit**

```bash
git add pkg/gcp services/api/internal/adapters/pubsub services/scraper/internal/adapters/gcp
git commit -m "refactor(eventos): adaptadores dependem de MessagePublisher"
```

---

### Task 2: Publicador HTTP (`pkg/pushhttp`)

**Files:**
- Create: `pkg/pushhttp/publisher.go`
- Test: `pkg/pushhttp/publisher_test.go`

**Interfaces:**
- Consumes: `gcp.MessagePublisher`, `gcp.PushEnvelope` (Tarefa 1 e `pkg/gcp/pubsub.go`).
- Produces: `pushhttp.New(baseURL string) *Publisher`; `(*Publisher).WithWaits(waits ...time.Duration) *Publisher`; `(*Publisher).Publish(ctx, topic string, data []byte, attrs map[string]string) (string, error)`.

- [ ] **Step 1: Write the failing tests**

`pkg/pushhttp/publisher_test.go`:

```go
package pushhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

var _ gcp.MessagePublisher = (*Publisher)(nil)

func TestPublishPostsPushEnvelopeToTopicRoute(t *testing.T) {
	var got gcp.PushEnvelope
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	id, err := New(srv.URL+"/").Publish(context.Background(), "gazette-fetched", []byte(`{"a":1}`), map[string]string{"type": "x"})

	if err != nil {
		t.Fatal(err)
	}
	if path != "/events/gazette-fetched" {
		t.Errorf("rota: %q", path)
	}
	if string(got.Message.Data) != `{"a":1}` || got.Message.Attributes["type"] != "x" {
		t.Errorf("envelope: %+v", got)
	}
	if id == "" || got.Message.MessageID != id {
		t.Errorf("id %q, messageId %q", id, got.Message.MessageID)
	}
	if got.Subscription != "push-http/gazette-fetched" {
		t.Errorf("subscription: %q", got.Subscription)
	}
}

func TestPublishRetriesServerErrorsAndRateLimit(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusInternalServerError)
		case 2:
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	_, err := New(srv.URL).WithWaits(0, 0, 0).Publish(context.Background(), "t", nil, nil)

	if err != nil || calls.Load() != 3 {
		t.Fatalf("err %v, chamadas %d", err, calls.Load())
	}
}

func TestPublishFailsAfterExhaustingRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	_, err := New(srv.URL).WithWaits(0, 0, 0).Publish(context.Background(), "t", nil, nil)

	if err == nil || calls.Load() != 4 {
		t.Fatalf("err %v, chamadas %d", err, calls.Load())
	}
}

func TestPublishDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	_, err := New(srv.URL).WithWaits(0, 0, 0).Publish(context.Background(), "t", nil, nil)

	if err == nil || calls.Load() != 1 {
		t.Fatalf("err %v, chamadas %d", err, calls.Load())
	}
}

func TestPublishRetriesWhenWorkerIsDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	_, err := New(url).WithWaits(0, 0, 0).Publish(context.Background(), "t", nil, nil)

	if err == nil {
		t.Fatal("worker fora do ar deveria virar erro")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd pkg && go test ./pushhttp/`
Expected: FAIL, pacote sem `New`.

- [ ] **Step 3: Implement**

`pkg/pushhttp/publisher.go`:

```go
package pushhttp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

type Publisher struct {
	baseURL string
	http    *http.Client
	waits   []time.Duration
}

func New(baseURL string) *Publisher {
	return &Publisher{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Minute},
		waits:   []time.Duration{time.Second, 4 * time.Second, 16 * time.Second},
	}
}

func (p *Publisher) WithWaits(waits ...time.Duration) *Publisher {
	c := *p
	c.waits = waits
	return &c
}

func (p *Publisher) Publish(ctx context.Context, topic string, data []byte, attrs map[string]string) (string, error) {
	id, err := messageID()
	if err != nil {
		return "", err
	}
	var env gcp.PushEnvelope
	env.Message.Data, env.Message.Attributes, env.Message.MessageID = data, attrs, id
	env.Subscription = "push-http/" + topic
	body, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	for attempt := 0; ; attempt++ {
		status, err := p.post(ctx, topic, body)
		if err == nil && status/100 == 2 {
			return id, nil
		}
		if err == nil && status != http.StatusTooManyRequests && status < 500 {
			return "", fmt.Errorf("push %s: status %d", topic, status)
		}
		if attempt >= len(p.waits) {
			return "", fmt.Errorf("push %s: %d tentativas: status %d: %v", topic, attempt+1, status, err)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(p.waits[attempt]):
		}
	}
}

func (p *Publisher) post(ctx context.Context, topic string, body []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/events/"+topic, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.http.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

func messageID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
```

- [ ] **Step 4: Run tests**

Run: `cd pkg && go test -race ./pushhttp/` e `make lint`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/pushhttp
git commit -m "feat(eventos): publicador que entrega por POST síncrono no worker"
```

---

### Task 3: Escolha do publicador por `EVENTS`

**Files:**
- Modify: `services/scraper/internal/config/config.go`, `services/scraper/internal/config/config_test.go`
- Modify: `services/api/internal/config/config.go`, teste de config da API (`services/api/internal/config/config_test.go`; crie se não existir)
- Modify: `services/scraper/cmd/scraper/main.go`, `services/api/cmd/worker/main.go`
- Create: `pkg/pushhttp/choose.go`, Test: `pkg/pushhttp/choose_test.go`

**Interfaces:**
- Consumes: `pushhttp.New` (Tarefa 2), `gcp.MessagePublisher` (Tarefa 1).
- Produces: campos `Events string` e `PushBaseURL string` nas duas `Config`; `pushhttp.Choose(events, baseURL string, pubsub gcp.MessagePublisher) (gcp.MessagePublisher, error)`; constantes `pushhttp.ModePubSub = "pubsub"` e `pushhttp.ModePushHTTP = "push-http"`.

- [ ] **Step 1: Write the failing tests**

`pkg/pushhttp/choose_test.go`:

```go
package pushhttp

import (
	"context"
	"testing"
)

type fakePubSub struct{}

func (fakePubSub) Publish(context.Context, string, []byte, map[string]string) (string, error) {
	return "ps", nil
}

func TestChooseDefaultsToPubSub(t *testing.T) {
	for _, mode := range []string{"", ModePubSub} {
		got, err := Choose(mode, "", fakePubSub{})
		if _, ok := got.(fakePubSub); err != nil || !ok {
			t.Errorf("%q: %T %v", mode, got, err)
		}
	}
}

func TestChoosePushHTTPNeedsBaseURL(t *testing.T) {
	if _, err := Choose(ModePushHTTP, "", fakePubSub{}); err == nil {
		t.Error("push-http sem PUSH_BASE_URL deveria falhar")
	}
	got, err := Choose(ModePushHTTP, "http://localhost:8081", fakePubSub{})
	if _, ok := got.(*Publisher); err != nil || !ok {
		t.Errorf("%T %v", got, err)
	}
}

func TestChooseRejectsUnknownMode(t *testing.T) {
	if _, err := Choose("pushhttp", "http://x", fakePubSub{}); err == nil {
		t.Error("modo desconhecido deveria falhar")
	}
}
```

No `services/scraper/internal/config/config_test.go`, acrescente:

```go
func TestLoadReadsEventsMode(t *testing.T) {
	t.Setenv("GCP_PROJECT_ID", "p")
	t.Setenv("GAZETTE_BUCKET", "b")
	t.Setenv("TOPIC_GAZETTE_FETCHED", "gazette-fetched")
	t.Setenv("TOPIC_FETCH_COMPLETED", "fetch-completed")
	t.Setenv("EVENTS", "push-http")
	t.Setenv("PUSH_BASE_URL", "http://localhost:8081")

	c, err := Load()

	if err != nil || c.Events != "push-http" || c.PushBaseURL != "http://localhost:8081" {
		t.Fatalf("%+v %v", c, err)
	}
}
```

E o equivalente para a API (papel worker), em `services/api/internal/config/config_test.go`:

```go
package config

import "testing"

func TestLoadWorkerReadsEventsMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("GCP_PROJECT_ID", "p")
	t.Setenv("GAZETTE_BUCKET", "b")
	t.Setenv("TOPIC_GAZETTE_INDEXED", "gazette-indexed")
	t.Setenv("EVENTS", "push-http")
	t.Setenv("PUSH_BASE_URL", "http://localhost:8081")

	c, err := Load(RoleWorker)

	if err != nil || c.Events != "push-http" || c.PushBaseURL != "http://localhost:8081" {
		t.Fatalf("%+v %v", c, err)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `make test`
Expected: FAIL (`undefined: Choose`, `c.Events undefined`).

- [ ] **Step 3: Implement**

`pkg/pushhttp/choose.go`:

```go
package pushhttp

import (
	"fmt"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

const (
	ModePubSub   = "pubsub"
	ModePushHTTP = "push-http"
)

func Choose(mode, baseURL string, pubsub gcp.MessagePublisher) (gcp.MessagePublisher, error) {
	switch mode {
	case "", ModePubSub:
		return pubsub, nil
	case ModePushHTTP:
		if baseURL == "" {
			return nil, fmt.Errorf("EVENTS=%s pede PUSH_BASE_URL", ModePushHTTP)
		}
		return New(baseURL), nil
	default:
		return nil, fmt.Errorf("EVENTS desconhecido: %q (use %s ou %s)", mode, ModePubSub, ModePushHTTP)
	}
}
```

Nas duas `Config`, ao lado de `PubSubEmulatorHost`: `Events string` e `PushBaseURL string`, lidos com `os.Getenv("EVENTS")` e `os.Getenv("PUSH_BASE_URL")` no `Load`.

`services/scraper/cmd/scraper/main.go`, no lugar do `gcp.NewEventPublisher(gcpclient.NewPublisher(…), …)`:

```go
	client, err := pushhttp.Choose(cfg.Events, cfg.PushBaseURL,
		gcpclient.NewPublisher(cfg.ProjectID, cfg.PubSubEmulatorHost, gcpclient.TokenSourceFor(cfg.PubSubEmulatorHost)))
	if err != nil {
		return err
	}
	publisher := gcp.NewEventPublisher(client, cfg.TopicFetched, cfg.TopicRuns)
```

(Mantenha os nomes de variável e o tratamento de erro do `main.go` atual; se a função não devolve `error`, siga o padrão de saída que ela já usa.)

`services/api/cmd/worker/main.go`, no lugar do `pubsub.NewEventPublisher(gcp.NewPublisher(…), cfg.TopicIndexed)`:

```go
	client, err := pushhttp.Choose(cfg.Events, cfg.PushBaseURL,
		gcp.NewPublisher(cfg.ProjectID, cfg.PubSubEmulatorHost, gcp.TokenSourceFor(cfg.PubSubEmulatorHost)))
	if err != nil {
		return err
	}
	publisher := pubsub.NewEventPublisher(client, cfg.TopicIndexed)
```

- [ ] **Step 4: Run tests**

Run: `make lint test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/pushhttp services/scraper services/api/internal/config services/api/cmd/worker
git commit -m "feat(eventos): EVENTS=push-http troca o Pub/Sub pelo POST no worker"
```

---

### Task 4: Integração do worker com o publicador HTTP

O worker publica `gazette-indexed` para ele mesmo e o handler de `gazette-fetched` só responde depois dos alertas.

**Files:**
- Create: `services/api/internal/integration/push_http_test.go`

**Interfaces:**
- Consumes: `pushhttp.New`, `pubsub.NewEventPublisher(gcp.MessagePublisher, string)`, `events.PushHandler`, os auxiliares de `e2e_test.go` (`openTestDB`, `textStore`, `passthroughExtractor`, `inbox`, `resetTables`, `post`).

- [ ] **Step 1: Write the test**

```go
//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	pkgevents "github.com/seu-usuario/diario-sg/pkg/events"
	"github.com/seu-usuario/diario-sg/pkg/pushhttp"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/email"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/pubsub"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
	"github.com/seu-usuario/diario-sg/services/api/internal/presentation/events"
	"github.com/seu-usuario/diario-sg/services/api/migrations"
)

func TestWorkerOverPushHTTPIndexesAndAlertsBeforeReturning(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	ctx := context.Background()
	db := openTestDB(t, ctx, url)
	defer db.Close()
	if _, err := postgres.Migrate(ctx, db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, resetTables); err != nil {
		t.Fatal(err)
	}
	gaz, acts := postgres.NewGazetteRepo(db), postgres.NewActRepo(db)
	subs, nlog := postgres.NewSubscriptionRepo(db), postgres.NewNotificationLog(db)
	box := &inbox{}
	notifier := email.NewNotifier(box, "https://web.exemplo")

	var handler http.Handler
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }))
	defer srv.Close()
	self := pushhttp.New(srv.URL)
	h := &events.PushHandler{
		Index: usecase.NewIndexGazette(gaz, textStore{}, passthroughExtractor{}, parser.Set{}, entities.New(),
			pubsub.NewEventPublisher(self, "gazette-indexed")),
		Match: usecase.NewMatchSubscriptions(gaz, acts, subs, nlog, notifier),
		Runs:  usecase.NewRecordFetchRun(postgres.NewFetchRunRepo(db)),
		Log:   slog.Default(),
	}
	handler = h.Routes()
	confirmSubscription(t, ctx, subs, box, "medicamentos")

	data, err := json.Marshal(pkgevents.GazetteFetched{EditionNumber: "1",
		PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC), SourceURL: "https://exemplo/1.pdf",
		StoragePath: "1.pdf", ChecksumSHA256: strings.Repeat("d", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := self.Publish(ctx, "gazette-fetched", data, map[string]string{pkgevents.AttrType: pkgevents.TypeGazetteFetched}); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM acts`).Scan(&count); err != nil || count == 0 {
		t.Fatalf("edição não indexada: %d %v", count, err)
	}
	if len(box.msgs) != 2 {
		t.Fatalf("esperava confirmação + alerta antes do Publish voltar, veio %d e-mails", len(box.msgs))
	}
}
```

`confirmSubscription` não existe: escreva-o no mesmo arquivo, repetindo o que o `TestEndToEnd` faz com a API HTTP (criar a assinatura e confirmar pelo token do primeiro e-mail). Leia `e2e_test.go` linhas 190–200 e use `postgres.SubscriptionRepo` direto se ele expuser criação e confirmação; senão, suba o `httpapi` como o `TestEndToEnd` e use `post`. A assinatura do auxiliar fica `func confirmSubscription(t *testing.T, ctx context.Context, subs *postgres.SubscriptionRepo, box *inbox, query string)`.

- [ ] **Step 2: Run**

Run: `make up && make test-integration`
Expected: PASS. Se falhar porque o `Publish` volta antes do alerta, confira que o `MatchSubscriptions` roda dentro do handler de `gazette-indexed` e que o `IndexGazette` publica antes de responder.

- [ ] **Step 3: Commit**

```bash
git add services/api/internal/integration/push_http_test.go
git commit -m "test(eventos): worker indexa e alerta pelo publicador HTTP"
```

---

### Task 5: Banco em 1 GB: `numeric_terms` no lugar do trigram do texto

**Files:**
- Create: `services/api/migrations/034_numeric_terms.sql`
- Modify: `services/api/internal/adapters/postgres/search.go`
- Create: `services/api/internal/integration/numeric_search_test.go`
- Modify: `docs/decisoes-de-codigo.md`

**Interfaces:**
- Produces: função SQL `numeric_terms(text) RETURNS text`, coluna `acts.numeric_terms`, índice `acts_numeric_terms_trgm_idx`; `matchClause` passa a usar `numeric_terms` para termo com dígito.

- [ ] **Step 1: Write the failing test**

`services/api/internal/integration/numeric_search_test.go` indexa uma edição com o texto abaixo pelo mesmo caminho do `TestEndToEnd` (`usecase.NewIndexGazette` com `textStore` trocado por um store que devolve este texto) e chama a busca da API (`acts.Search` com `domain.ActFilter{Query: …}`, como o `name_search_test.go` faz). Texto:

```
EXTRATO DO CONTRATO Nº 55/2026
Processo nº65/100410/2018. Contratada: Construtora Alfa LTDA, CNPJ 28.636.579/0001-00.
PORTARIA Nº 9/2026
Nomeia servidor da Secretaria de Obras.
```

Casos (nome do subteste → termo → quantos atos):

```go
cases := []struct{ name, q string; want int }{
	{"processo parcial colado em outro texto", "100410/2018", 1},
	{"cnpj formatado", "28.636.579/0001-00", 1},
	{"percentual no meio do número é literal", "100%410", 0},
	{"texto sem dígito pelo full-text", "construtora", 1},
	{"trecho de palavra sem dígito não casa mais", "nstrutora alf", 0},
}
```

E um teste de plano:

```go
func TestDigitSearchUsesNumericTermsIndex(t *testing.T) {
	// após indexar a edição acima e rodar ANALYZE acts:
	var plan strings.Builder
	rows, err := db.QueryContext(ctx, `EXPLAIN SELECT id FROM acts WHERE numeric_terms ILIKE '%100410/2018%'`)
	// junte as linhas em plan
	if strings.Contains(plan.String(), "Seq Scan on acts") {
		t.Fatalf("termo com dígito não pode varrer acts:\n%s", plan.String())
	}
}
```

(Escreva o corpo completo seguindo o padrão de `name_search_test.go`: `openTestDB`, `postgres.Migrate`, `resetTables`, indexar, consultar. Antes do `EXPLAIN`, rode `SET enable_seqscan = off` só se a tabela tiver poucas linhas e o planejador preferir varrer mesmo com o índice; o objetivo é provar que o índice serve à consulta.)

- [ ] **Step 2: Run to verify it fails**

Run: `make test-integration`
Expected: FAIL: `column "numeric_terms" does not exist` e "trecho de palavra" casando 1.

- [ ] **Step 3: Write the migration**

`services/api/migrations/034_numeric_terms.sql`:

```sql
CREATE FUNCTION numeric_terms(t text) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS
$$ SELECT coalesce(string_agg(m[1], ' '), '') FROM regexp_matches(t, '([0-9][0-9./-]*[0-9])', 'g') AS m $$;

ALTER TABLE acts ADD COLUMN numeric_terms text GENERATED ALWAYS AS (numeric_terms(body)) STORED;

CREATE INDEX acts_numeric_terms_trgm_idx ON acts USING gin (numeric_terms gin_trgm_ops);

DROP INDEX acts_body_trgm_idx;
```

- [ ] **Step 4: Change the search**

Em `search.go`, `matchClause` vira:

```go
const matchClause = `(a.search @@ q
		  OR ($Q ~ '[0-9]' AND a.numeric_terms ILIKE $LIKE))`
```

e remova `minSubstringRunes`. O `likePattern` já escapa `%`, `_` e `\`. O `$LIKE` não passa mais por `unaccent_immutable`: números não têm acento.

Rode `gofmt` e confira que nada mais usa `minSubstringRunes` (`grep -rn minSubstringRunes services/`).

- [ ] **Step 5: Run tests**

Run: `make lint test && make test-integration`
Expected: PASS em tudo, inclusive nos testes de busca que já existiam. Se algum teste antigo dependia de substring sem dígito, ajuste-o para refletir a decisão e cite isso no commit.

- [ ] **Step 6: Record the decision**

Em `docs/decisoes-de-codigo.md`, na seção de busca, acrescente um parágrafo: por que o trigram saiu (224 MB, limite de 1 GB do Neon grátis), o que a coluna `numeric_terms` guarda e as contagens medidas na base local (da spec, seção 4): `100410/2018` 1 → 1; `28.636.579/0001-00` 5.289 → 5.289; "Construtora" 433 → 431; "secretaria municipal de saude" 14.026 → 14.022; "dispensa de licitacao" 946 → 946.

- [ ] **Step 7: Commit**

```bash
git add services/api/migrations/034_numeric_terms.sql services/api/internal/adapters/postgres/search.go services/api/internal/integration/numeric_search_test.go docs/decisoes-de-codigo.md
git commit -m "feat(busca): trechos numéricos com índice próprio no lugar do trigram do texto"
```

---

### Task 6: IP do cliente atrás do proxy (`TRUSTED_PROXY_HOPS`)

**Files:**
- Modify: `services/api/internal/presentation/http/client.go`, `client_test.go`
- Modify: chamadores de `clientKey` e `perClient` (`mcp_keys.go`, `oauth.go`, `router.go`; ache com `grep -rn "clientKey\|perClient" services/api/internal/presentation/http`)
- Modify: `services/api/internal/presentation/http/router.go` (campo `ProxyHops int` em `API`)
- Modify: `services/api/internal/config/config.go` (`TrustedProxyHops int`, de `TRUSTED_PROXY_HOPS`, padrão 0; valor inválido ou negativo vira erro de configuração)
- Modify: `services/api/cmd/api/main.go` (passa `cfg.TrustedProxyHops` para `API.ProxyHops`)

**Interfaces:**
- Produces: `clientKey(r *http.Request, hops int) string`; `(a *API) perClient(limiter *ratelimit.Limiter, next http.Handler) http.Handler`.

- [ ] **Step 1: Write the failing test**

Substitua `client_test.go` por:

```go
package http

import (
	"net/http/httptest"
	"testing"
)

func TestClientKey(t *testing.T) {
	cases := []struct {
		name, xff string
		hops      int
		want      string
	}{
		{"sem cabeçalho usa a conexão", "", 1, "10.0.0.1"},
		{"hops 0 mantém o primeiro endereço", " 200.1.2.3 , 10.0.0.9", 0, "200.1.2.3"},
		{"hops 1 pega o que o proxy acrescentou", "1.1.1.1, 200.1.2.3", 1, "200.1.2.3"},
		{"hops 2 pula um proxy confiável", "1.1.1.1, 200.1.2.3, 10.0.0.9", 2, "200.1.2.3"},
		{"menos entradas que hops cai na conexão", "200.1.2.3", 3, "10.0.0.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/v1/reports", nil)
			r.RemoteAddr = "10.0.0.1:5555"
			if c.xff != "" {
				r.Header.Set("X-Forwarded-For", c.xff)
			}
			if got := clientKey(r, c.hops); got != c.want {
				t.Errorf("veio %q, queria %q", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/api && go test ./internal/presentation/http/ -run TestClientKey`
Expected: FAIL (assinatura antiga).

- [ ] **Step 3: Implement**

```go
func clientKey(r *http.Request, hops int) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		switch {
		case hops == 0:
			return strings.TrimSpace(parts[0])
		case hops <= len(parts):
			return strings.TrimSpace(parts[len(parts)-hops])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
```

Cada `clientKey(r)` vira `clientKey(r, a.ProxyHops)`; `perClient` vira método de `*API` para ler `a.ProxyHops`, e as chamadas `perClient(…)` viram `a.perClient(…)`.

- [ ] **Step 4: Run tests**

Run: `make lint test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/api
git commit -m "feat(api): limite por IP lê o X-Forwarded-For na posição do proxy confiável"
```

---

### Task 7: PDF oficial quando não há bucket

**Files:**
- Modify: `services/api/internal/core/ports/ports.go` (porta `SourcePDF`)
- Create: `services/api/internal/core/usecase/open_gazette_pdf.go`, Test: `open_gazette_pdf_test.go`
- Modify: `services/api/internal/core/usecase/gazette_pdf.go`, `read_page.go` e os testes `gazette_pdf_test.go`, `read_page_test.go`
- Create: `services/api/internal/adapters/sourcepdf/fetcher.go`, Test: `fetcher_test.go`
- Create: `services/api/internal/adapters/nostore/store.go`, Test: `store_test.go`
- Modify: `services/api/internal/config/config.go` (API sem `GAZETTE_BUCKET` obrigatório)
- Modify: `services/api/cmd/api/main.go`
- Modify: `services/api/internal/integration/organ_test.go`, `investigator_test.go` (novos parâmetros)

**Interfaces:**
- Produces: `ports.SourcePDF interface { Open(ctx context.Context, url string) (io.ReadCloser, error) }`; `usecase.NewGetGazettePDF(g ports.GazetteRepository, s ports.FileStorage, src ports.SourcePDF)`; `usecase.NewReadPage(g ports.GazetteRepository, s ports.FileStorage, src ports.SourcePDF, p ports.PageExtractor)`; `sourcepdf.New() *Fetcher`; `nostore.Store{}` com `Get`, `Put` e `Exists`.

- [ ] **Step 1: Write the failing tests**

`open_gazette_pdf_test.go`:

```go
package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type failingStorage struct{}

func (failingStorage) Get(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("sem objeto")
}

type recordingSource struct {
	urls []string
	err  error
}

func (s *recordingSource) Open(_ context.Context, url string) (io.ReadCloser, error) {
	s.urls = append(s.urls, url)
	if s.err != nil {
		return nil, s.err
	}
	return io.NopCloser(strings.NewReader("%PDF-oficial")), nil
}

func TestOpenGazettePDFFallsBackToSourceURL(t *testing.T) {
	src := &recordingSource{}
	g := domain.Gazette{ID: "g1", StoragePath: "1.pdf", SourceURL: "https://oficial/1.pdf"}

	body, err := openGazettePDF(context.Background(), failingStorage{}, src, g)

	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(body)
	if string(b) != "%PDF-oficial" || len(src.urls) != 1 || src.urls[0] != g.SourceURL {
		t.Errorf("corpo %q, urls %v", b, src.urls)
	}
}

func TestOpenGazettePDFPrefersStorage(t *testing.T) {
	src := &recordingSource{}

	_, err := openGazettePDF(context.Background(), memStorage{}, src, domain.Gazette{ID: "g1", StoragePath: "1.pdf", SourceURL: "https://oficial/1.pdf"})

	if err != nil || len(src.urls) != 0 {
		t.Fatalf("não devia ir à fonte: %v %v", err, src.urls)
	}
}

func TestOpenGazettePDFWithoutSourceURLFails(t *testing.T) {
	_, err := openGazettePDF(context.Background(), failingStorage{}, &recordingSource{}, domain.Gazette{ID: "g1"})

	if err == nil {
		t.Fatal("sem storage e sem source_url deveria falhar")
	}
}
```

(`memStorage` já existe nos testes de usecase e devolve um corpo para qualquer caminho; confira em `gazette_pdf_test.go` e, se o comportamento for outro, use um fake local.)

`sourcepdf/fetcher_test.go`:

```go
package sourcepdf

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenReturnsPDFBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("sem User-Agent")
		}
		w.Header().Set("Content-Type", "application/pdf")
		io.WriteString(w, "%PDF-1.4 conteúdo")
	}))
	defer srv.Close()

	body, err := New().Open(context.Background(), srv.URL)

	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	b, _ := io.ReadAll(body)
	if string(b) != "%PDF-1.4 conteúdo" {
		t.Errorf("%q", b)
	}
}

func TestOpenRejectsNonOKAndNonPDF(t *testing.T) {
	cases := map[string]http.HandlerFunc{
		"404":             func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		"html com 200":    func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "<html>erro</html>") },
		"corpo vazio 200": func(w http.ResponseWriter, _ *http.Request) {},
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(h)
			defer srv.Close()
			if body, err := New().Open(context.Background(), srv.URL); err == nil {
				body.Close()
				t.Fatal("devia falhar")
			}
		})
	}
}
```

`nostore/store_test.go`:

```go
package nostore

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

func TestStoreHasNothing(t *testing.T) {
	ctx := context.Background()
	if _, err := (Store{}).Get(ctx, "x"); !errors.Is(err, gcp.ErrObjectNotFound) {
		t.Errorf("Get: %v", err)
	}
	if ok, err := (Store{}).Exists(ctx, "x"); ok || err != nil {
		t.Errorf("Exists: %v %v", ok, err)
	}
	if err := (Store{}).Put(ctx, "x", "text/plain", nil); err != nil {
		t.Errorf("Put: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `make test`
Expected: FAIL (símbolos inexistentes).

- [ ] **Step 3: Implement**

`ports.go`, ao lado de `FileStorage`:

```go
type SourcePDF interface {
	Open(ctx context.Context, url string) (io.ReadCloser, error)
}
```

`usecase/open_gazette_pdf.go`:

```go
package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

func openGazettePDF(ctx context.Context, s ports.FileStorage, src ports.SourcePDF, g domain.Gazette) (io.ReadCloser, error) {
	body, err := s.Get(ctx, g.StoragePath)
	if err == nil {
		return body, nil
	}
	if g.SourceURL == "" || src == nil {
		return nil, fmt.Errorf("pdf da edição %s: %w", g.ID, err)
	}
	body, srcErr := src.Open(ctx, g.SourceURL)
	if srcErr != nil {
		return nil, fmt.Errorf("pdf da edição %s: storage: %v; fonte: %w", g.ID, err, srcErr)
	}
	return body, nil
}
```

`gazette_pdf.go`: struct ganha `source ports.SourcePDF`; `NewGetGazettePDF(g, s, src)`; `Open` vira `return openGazettePDF(ctx, uc.storage, uc.source, g)`.

`read_page.go`: struct ganha `source ports.SourcePDF`; `NewReadPage(g, s, src, p)`; troque o `uc.storage.Get(...)` e o `if err` seguinte por `body, err := openGazettePDF(ctx, uc.storage, uc.source, g)` e `if err != nil { return PageText{}, err }`.

Atualize os chamadores: nos testes de usecase passe `nil` como `src` (o comportamento antigo); em `organ_test.go` e `investigator_test.go` também `nil`.

`sourcepdf/fetcher.go`:

```go
package sourcepdf

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const userAgent = "diario-sg-bot/1.0 (+https://github.com/Lucantas/diario-sg)"

type Fetcher struct{ http *http.Client }

func New() *Fetcher { return &Fetcher{http: &http.Client{Timeout: 2 * time.Minute}} }

func (f *Fetcher) Open(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("pdf oficial %s: status %d", url, resp.StatusCode)
	}
	br := bufio.NewReader(resp.Body)
	head, _ := br.Peek(5)
	if !bytes.HasPrefix(head, []byte("%PDF-")) {
		resp.Body.Close()
		return nil, fmt.Errorf("pdf oficial %s: resposta não é PDF", url)
	}
	return struct {
		io.Reader
		io.Closer
	}{br, resp.Body}, nil
}
```

Confira o User-Agent que o scraper usa (`grep -rn User-Agent services/scraper`) e use o mesmo texto.

`nostore/store.go`:

```go
package nostore

import (
	"context"
	"fmt"
	"io"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

type Store struct{}

func (Store) Get(_ context.Context, name string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("storage get %q: %w", name, gcp.ErrObjectNotFound)
}

func (Store) Exists(context.Context, string) (bool, error) { return false, nil }

func (Store) Put(context.Context, string, string, io.Reader) error { return nil }
```

Confira que `ocrcache.New` aceita esse tipo (a interface que ele recebe deve ter `Get`, `Put` e, se for o caso, `Exists`); se pedir outro método, acrescente-o ao `Store` com o mesmo espírito (nada guardado).

`config.go`: tire `RoleAPI` da lista que exige `GAZETTE_BUCKET`.

`cmd/api/main.go`:

```go
	var storage interface {
		Get(context.Context, string) (io.ReadCloser, error)
		Put(context.Context, string, string, io.Reader) error
		Exists(context.Context, string) (bool, error)
	} = nostore.Store{}
	if cfg.Bucket != "" {
		storage = gcp.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))
	}
	source := sourcepdf.New()
```

(Se o `ocrcache.New` pedir um tipo nomeado, use esse tipo na declaração.) Passe `source` em `NewReadPage(gazettes, storage, source, pdf.New().WithCache(ocrcache.New(storage).ReadOnly(), l))` e `NewGetGazettePDF(gazettes, storage, source)`.

- [ ] **Step 4: Run tests**

Run: `make lint test && make test-integration`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/api
git commit -m "feat(api): PDF da edição e leitura de página caem no PDF oficial sem bucket"
```

---

### Task 8: Dump em diretório (`DUMP_DIR`)

**Files:**
- Create: `services/api/internal/adapters/fsstore/store.go`, Test: `store_test.go`
- Modify: `services/api/internal/config/config.go` (`DumpDir string` de `DUMP_DIR`; papel dump exige `DUMPS_BUCKET` **ou** `DUMP_DIR`)
- Modify: `services/api/cmd/dump/main.go`

**Interfaces:**
- Consumes: `ports.ObjectWriter` (`Put(ctx, name, contentType string, body io.Reader) error`).
- Produces: `fsstore.New(dir string) *Store`.

- [ ] **Step 1: Write the failing test**

```go
package fsstore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPutWritesFileUnderDir(t *testing.T) {
	dir := t.TempDir()

	err := New(dir).Put(context.Background(), "latest/manifest.json", "application/json", strings.NewReader(`{"ok":true}`))

	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "latest", "manifest.json"))
	if err != nil || string(b) != `{"ok":true}` {
		t.Fatalf("%q %v", b, err)
	}
}

func TestPutRefusesPathsOutsideDir(t *testing.T) {
	if err := New(t.TempDir()).Put(context.Background(), "../fora.txt", "text/plain", strings.NewReader("x")); err == nil {
		t.Fatal("caminho fora do diretório deveria falhar")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/api && go test ./internal/adapters/fsstore/`
Expected: FAIL.

- [ ] **Step 3: Implement**

```go
package fsstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Store struct{ dir string }

func New(dir string) *Store { return &Store{dir: dir} }

func (s *Store) Put(_ context.Context, name, _ string, body io.Reader) error {
	path := filepath.Join(s.dir, filepath.FromSlash(name))
	rel, err := filepath.Rel(s.dir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("caminho fora do diretório do dump: %q", name)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
```

`cmd/dump/main.go`: escolha o destino.

```go
	var objects ports.ObjectWriter = gcp.NewStorage(cfg.DumpsBucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))
	if cfg.DumpDir != "" {
		objects = fsstore.New(cfg.DumpDir)
	}
	manifest, err := usecase.NewPublishDump(postgres.NewDumpSource(db), objects).Execute(ctx, start.UTC())
```

Config: no papel dump, troque o `required["DUMPS_BUCKET"]` por um erro quando os dois estiverem vazios (`"DUMPS_BUCKET ou DUMP_DIR"`), com teste em `config_test.go`.

- [ ] **Step 4: Run tests**

Run: `make lint test && make test-integration`
Expected: PASS (o `dump_test.go` de integração continua usando o bucket ou o fake que já usa).

- [ ] **Step 5: Commit**

```bash
git add services/api
git commit -m "feat(dump): DUMP_DIR grava o dump em diretório"
```

---

### Task 9: Workflows do GitHub Actions

**Files:**
- Create: `scripts/ci-gcs.sh`
- Create: `.github/workflows/_pipeline.yml`, `.github/workflows/diario-prefeitura.yml`, `.github/workflows/diario-camara.yml`
- Create: `.github/workflows/_carga.yml`, `.github/workflows/carga-{sicam,sancoes,tce,pncp,federal,agentes,receita}.yml`
- Create: `.github/workflows/dump.yml`
- Modify: `.github/workflows/deploy.yml`, `infra.yml`, `reindex.yml` (gatilhos só `workflow_dispatch`)
- Modify: `.github/workflows/ci.yml` (job `actionlint`)

**Interfaces:**
- Consumes: `EVENTS`, `PUSH_BASE_URL` (Tarefa 3), `DUMP_DIR` (Tarefa 8), os binários de `services/api/cmd/*` e `services/scraper/cmd/scraper`.
- Segredos do ambiente `producao`: `DATABASE_URL`, `RESEND_API_KEY`, `EMAIL_FROM`, `RECEITA_SHARE_TOKEN` (se as cargas exigirem; confira com `grep -rn Getenv services/api/internal/config`). Variáveis: `PUBLIC_WEB_URL`.

- [ ] **Step 1: Script do emulador GCS**

`scripts/ci-gcs.sh` (executável):

```bash
#!/usr/bin/env bash
set -euo pipefail

DATA="${1:?uso: ci-gcs.sh <diretório de dados>}"
mkdir -p "$DATA/diario-gazettes"
docker run -d --name gcs -p 4443:4443 -v "$DATA:/data" fsouza/fake-gcs-server:latest \
  -scheme http -port 4443 -public-host localhost:4443 -backend filesystem -filesystem-root /data >/dev/null
for _ in $(seq 1 30); do
  curl -fsS http://localhost:4443/storage/v1/b >/dev/null 2>&1 && exit 0
  sleep 1
done
echo "emulador GCS não subiu" >&2
exit 1
```

O bucket existe porque o diretório `diario-gazettes` já está em `/data` (o backend de arquivos lê os buckets das pastas). Confira rodando local: `scripts/ci-gcs.sh /tmp/gcs-teste && curl -s localhost:4443/storage/v1/b` deve listar `diario-gazettes`; depois `docker rm -f gcs`.

- [ ] **Step 2: Pipeline reutilizável**

`.github/workflows/_pipeline.yml`:

```yaml
name: pipeline

on:
  workflow_call:
    inputs:
      source:
        required: true
        type: string
      lookback_days:
        required: false
        type: string
        default: "3"

permissions:
  contents: read

jobs:
  coleta:
    runs-on: ubuntu-latest
    timeout-minutes: 60
    environment: producao
    env:
      DATABASE_URL: ${{ secrets.DATABASE_URL }}
      RESEND_API_KEY: ${{ secrets.RESEND_API_KEY }}
      EMAIL_FROM: ${{ secrets.EMAIL_FROM }}
      NOTIFIER: ${{ secrets.RESEND_API_KEY != '' && 'resend' || 'log' }}
      PUBLIC_WEB_URL: ${{ vars.PUBLIC_WEB_URL }}
      GCP_PROJECT_ID: local
      GAZETTE_BUCKET: diario-gazettes
      STORAGE_EMULATOR_HOST: localhost:4443
      TOPIC_GAZETTE_FETCHED: gazette-fetched
      TOPIC_GAZETTE_INDEXED: gazette-indexed
      TOPIC_FETCH_COMPLETED: fetch-completed
      EVENTS: push-http
      PUSH_BASE_URL: http://localhost:8081
      SOURCE: ${{ inputs.source }}
      LOOKBACK_DAYS: ${{ inputs.lookback_days }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: services/api/go.mod
          cache-dependency-path: '**/go.sum'
      - name: OCR e poppler
        run: sudo apt-get update && sudo apt-get install -y --no-install-recommends poppler-utils tesseract-ocr tesseract-ocr-por
      - name: compila
        run: |
          mkdir -p bin
          (cd services/api && go build -o ../../bin/worker ./cmd/worker)
          (cd services/scraper && go build -o ../../bin/scraper ./cmd/scraper)
      - name: cache do OCR
        uses: actions/cache@v4
        with:
          path: gcs/diario-gazettes/ocr
          key: ocr-${{ github.run_id }}
          restore-keys: ocr-
      - name: emulador GCS
        run: scripts/ci-gcs.sh "$PWD/gcs"
      - name: worker
        run: |
          PORT=8081 nohup bin/worker > worker.log 2>&1 &
          for _ in $(seq 1 30); do curl -fsS localhost:8081/healthz && exit 0; sleep 1; done
          cat worker.log; exit 1
      - name: scraper
        run: bin/scraper
      - name: log do worker
        if: always()
        run: cat worker.log
```

Confira o nome da variável de porta e da saúde do worker (`PORT`, `/healthz`) no `services/api/cmd/worker/main.go` e no `PushHandler`. Confira se o caminho do cache do OCR no emulador é `gcs/diario-gazettes/ocr` (o prefixo é `ocr/` em `adapters/ocrcache/store.go`).

- [ ] **Step 3: As duas fontes**

`.github/workflows/diario-prefeitura.yml`:

```yaml
name: diario-prefeitura

on:
  schedule:
    - cron: "0 11,16,22 * * 1-6"
  workflow_dispatch:
    inputs:
      lookback_days:
        description: Dias para trás
        default: "3"

concurrency:
  group: diario-prefeitura

jobs:
  pipeline:
    uses: ./.github/workflows/_pipeline.yml
    with:
      source: diario_prefeitura
      lookback_days: ${{ inputs.lookback_days || '3' }}
    secrets: inherit
```

`.github/workflows/diario-camara.yml`: igual, com `name: diario-camara`, `cron: "30 0 * * 0,2-6"`, `group: diario-camara` e `source: diario_camara`. Confira o valor de `SOURCE` da Câmara em `services/scraper/internal/core/domain` (o `make run-scraper-camara` usa `diario_camara`).

- [ ] **Step 4: Cargas auxiliares**

`.github/workflows/_carga.yml`:

```yaml
name: carga

on:
  workflow_call:
    inputs:
      cmd:
        required: true
        type: string
      args:
        required: false
        type: string
        default: ""
      timeout:
        required: false
        type: number
        default: 60

permissions:
  contents: read

jobs:
  carga:
    runs-on: ubuntu-latest
    timeout-minutes: ${{ inputs.timeout }}
    environment: producao
    env:
      DATABASE_URL: ${{ secrets.DATABASE_URL }}
      RECEITA_SHARE_TOKEN: ${{ secrets.RECEITA_SHARE_TOKEN }}
      GCP_PROJECT_ID: local
      GAZETTE_BUCKET: diario-gazettes
      STORAGE_EMULATOR_HOST: localhost:4443
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: services/api/go.mod
          cache-dependency-path: '**/go.sum'
      - name: emulador GCS
        run: scripts/ci-gcs.sh "$PWD/gcs"
      - name: ${{ inputs.cmd }}
        working-directory: services/api
        run: go run ./cmd/${{ inputs.cmd }} ${{ inputs.args }}
```

Um arquivo por carga, todos no formato abaixo (troque nome, cron, cmd, args e timeout pela tabela):

```yaml
name: carga-tce

on:
  schedule:
    - cron: "0 8 * * 0"
  workflow_dispatch:

concurrency:
  group: carga-tce

jobs:
  carga:
    uses: ./.github/workflows/_carga.yml
    with:
      cmd: tce
      args: "-from 0 -to 0"
      timeout: 60
    secrets: inherit
```

| arquivo | cron | cmd | args | timeout |
| --- | --- | --- | --- | --- |
| carga-sicam.yml | `0 8 * * *` | sicam | `-max 3000` | 60 |
| carga-sancoes.yml | `0 10 * * *` | sancoes | | 30 |
| carga-tce.yml | `0 8 * * 0` | tce | `-from 0 -to 0` | 60 |
| carga-pncp.yml | `0 9 * * 0` | pncp | `-from 0 -to 0` | 60 |
| carga-federal.yml | `0 11 * * 0` | federal | | 30 |
| carga-agentes.yml | `0 12 10 * *` | agentes | | 30 |
| carga-receita.yml | `0 9 20 * *` | receita | | 360 |

Os argumentos repetem os padrões do `Makefile` (`-from 0 -to 0` é o "padrão" do `tce` e do `pncp`; `federal`, `agentes` e `receita` sem flags usam o padrão deles). Confira cada um no `Makefile` e no `cmd/<carga>/main.go`.

- [ ] **Step 5: Dump e tamanho do banco**

`.github/workflows/dump.yml`:

```yaml
name: dump

on:
  schedule:
    - cron: "0 7 * * 0"
  workflow_dispatch:

concurrency:
  group: dump

permissions:
  contents: write

jobs:
  dump:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    environment: producao
    env:
      DATABASE_URL: ${{ secrets.DATABASE_URL }}
      DUMP_DIR: ${{ github.workspace }}/out
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: services/api/go.mod
          cache-dependency-path: '**/go.sum'
      - name: dump
        working-directory: services/api
        run: go run ./cmd/dump
      - name: publica no branch dados
        run: |
          cd out/latest
          git init -q -b dados
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git add -A
          git commit -q -m "dados: dump de $(date -u +%F)"
          git push -f "https://x-access-token:${{ github.token }}@github.com/${{ github.repository }}.git" dados
      - name: tamanho do banco
        run: |
          size=$(psql "$DATABASE_URL" -Atc "select pg_database_size(current_database())")
          echo "banco: $((size / 1024 / 1024)) MB"
          if [ "$size" -gt $((900 * 1024 * 1024)) ]; then
            echo "::error::banco acima de 900 MB; o limite do Neon grátis é 1 GB"
            exit 1
          fi
```

(O `ubuntu-latest` já traz o `psql`.)

- [ ] **Step 6: Desligar os workflows do GCP**

Em `deploy.yml`, `infra.yml` e `reindex.yml`, troque o bloco `on:` por:

```yaml
on:
  workflow_dispatch:
```

- [ ] **Step 7: actionlint no CI**

Em `ci.yml`, acrescente o job:

```yaml
  actionlint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: actionlint
        run: |
          bash <(curl -fsSL https://raw.githubusercontent.com/rhysd/actionlint/main/scripts/download-actionlint.bash)
          ./actionlint -color
```

Rode o mesmo local antes de commitar: baixe o binário em `/tmp` com o mesmo script e rode `/tmp/actionlint` na raiz do repo.
Expected: sem erros.

- [ ] **Step 8: Commit**

```bash
git add scripts/ci-gcs.sh .github/workflows
git commit -m "ci: coleta, cargas e dump agendados no GitHub Actions"
```

---

### Task 10: Blueprint do Render

**Files:**
- Create: `render.yaml`

- [ ] **Step 1: Escrever o blueprint**

```yaml
services:
  - type: web
    name: diario-sg-api
    runtime: docker
    plan: free
    region: ohio
    dockerfilePath: ./services/api/Dockerfile
    dockerContext: .
    healthCheckPath: /healthz
    autoDeploy: true
    branch: main
    envVars:
      - key: DATABASE_URL
        sync: false
      - key: PUBLIC_WEB_URL
        sync: false
      - key: NOTIFIER
        value: resend
      - key: RESEND_API_KEY
        sync: false
      - key: EMAIL_FROM
        sync: false
      - key: TRUSTED_PROXY_HOPS
        value: "1"

  - type: web
    name: diario-sg-web
    runtime: static
    rootDir: apps/web
    buildCommand: npm ci && npm run build
    staticPublishPath: dist
    autoDeploy: true
    branch: main
    routes:
      - type: rewrite
        source: /api/*
        destination: https://diario-sg-api.onrender.com/*
      - type: rewrite
        source: /.well-known/*
        destination: https://diario-sg-api.onrender.com/.well-known/*
      - type: rewrite
        source: /dados/*
        destination: https://raw.githubusercontent.com/Lucantas/diario-sg/dados/*
      - type: rewrite
        source: /*
        destination: /index.html
```

Confira, antes de commitar:
- a rota de saúde da API (`grep -rn healthz services/api/internal/presentation/http`). Se a API não tiver `/healthz`, use uma rota leve que exista (ex.: `/v1/...`) ou acrescente `GET /healthz` ao router com teste;
- se o `nginx` do web reescreve `/api/` tirando o prefixo (`proxy_pass ${API_URL}/;`): sim, então `/api/*` → `…onrender.com/*` está certo;
- se o nginx manda `/.well-known/oauth-*` sem tirar o prefixo: sim (`proxy_pass ${API_URL};`), então a reescrita mantém `/.well-known/`;
- se a API recusa variáveis que o blueprint não define (`GCP_PROJECT_ID`, tópicos): o papel `api` só exige `DATABASE_URL` depois da Tarefa 7; confirme com `DATABASE_URL=postgres://x go run ./cmd/api` local, que deve passar da configuração e só falhar na conexão.

- [ ] **Step 2: Validar o YAML**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('render.yaml'))"`
Expected: sem erro.

- [ ] **Step 3: Commit**

```bash
git add render.yaml
git commit -m "feat(deploy): blueprint do Render para a API e o site"
```

---

### Task 11: Runbook e roadmap

**Files:**
- Create: `docs/deploy-gratis.md`
- Modify: `docs/roadmap.md` (lista de controle das pendências)
- Modify: `README.md` (seção de deploy, um parágrafo apontando para o runbook)

- [ ] **Step 1: Runbook**

`docs/deploy-gratis.md`, com estas seções e os comandos de cada uma:

1. **Neon**: projeto `diario-sg` na região AWS US East 2 (Ohio), Postgres 16; em Compute, autoscaling de 0,25 a 0,25 CU e suspensão após 5 min; copiar a connection string *pooled* com `sslmode=require`.
2. **Carga inicial** (na máquina com a base local):
   ```bash
   make up && make migrate
   docker compose exec postgres psql -U postgres -d diario -c "VACUUM FULL ANALYZE"
   docker compose exec postgres psql -U postgres -d diario -Atc "select pg_size_pretty(pg_database_size('diario'))"
   docker compose exec postgres pg_dump -U postgres -d diario -Fc --no-owner --no-privileges -f /tmp/diario.dump
   docker compose cp postgres:/tmp/diario.dump ./diario.dump
   pg_restore --no-owner --no-privileges -d "$NEON_DATABASE_URL" ./diario.dump
   ```
   O `pg_restore` cria as extensões (`pg_trgm`, `unaccent`) que o Neon suporta; se alguma falhar por permissão, criar antes com `CREATE EXTENSION` no SQL Editor do Neon.
3. **GitHub**: ambiente `producao` em Settings → Environments; segredos `DATABASE_URL`, `RESEND_API_KEY`, `EMAIL_FROM`, `RECEITA_SHARE_TOKEN`; variável `PUBLIC_WEB_URL`. Pela linha de comando: `gh secret set DATABASE_URL -R Lucantas/diario-sg -e producao` (cola o valor quando pedir).
4. **Render**: New → Blueprint → este repositório; preencher as variáveis `sync: false`; depois do primeiro deploy, conferir `https://diario-sg-api.onrender.com/healthz`.
5. **Domínio**: CNAME do domínio (ou `www`) para o `diario-sg-web.onrender.com` e o domínio adicionado em Settings → Custom Domains do site; no Resend, adicionar o domínio e criar no DNS os registros que ele mostrar; `EMAIL_FROM` com esse domínio; `PUBLIC_WEB_URL` com a URL final.
6. **Limite por IP**: com o site no ar, fazer uma requisição pelo domínio e olhar no log da API o `X-Forwarded-For` que chega; ajustar `TRUSTED_PROXY_HOPS` para a posição do IP do cliente contando da direita.
7. **Rodar à mão**: `gh workflow run diario-prefeitura -R Lucantas/diario-sg -f lookback_days=7` e o equivalente para as cargas.
8. **Quando o banco passar de 900 MB**: o workflow `dump` falha; opções: `VACUUM FULL` no Neon, arquivar edições antigas, ou o plano pago.

- [ ] **Step 2: Roadmap e README**

Na "Lista de controle das pendências" do `docs/roadmap.md`: o item "Nuvem: o environment `dev` …" ganha a linha "Deploy grátis (GitHub Actions + Render + Neon) em `docs/deploy-gratis.md`; o GCP fica para quando houver conta", e o subitem "Limites por IP" passa a apontar para `TRUSTED_PROXY_HOPS`. No `README.md`, na parte de deploy, um parágrafo dizendo que o deploy ativo é o do runbook e que o Terraform descreve o GCP.

- [ ] **Step 3: Commit**

```bash
git add docs/deploy-gratis.md docs/roadmap.md README.md
git commit -m "docs(deploy): runbook do deploy grátis"
```

---

### Task 12: Banco local pronto para o Neon

Não tem código: prepara e mede a base que vai para o Neon. Depende das Tarefas 5 e 7 aplicadas.

- [ ] **Step 1: Aplicar e compactar**

```bash
make up && make migrate
docker compose exec postgres psql -U postgres -d diario -c "VACUUM FULL ANALYZE"
docker compose exec postgres psql -U postgres -d diario -Atc "select pg_size_pretty(pg_database_size('diario'))"
```

Expected: abaixo de 900 MB. Se passar, pare e registre o número na spec antes de seguir.

- [ ] **Step 2: Conferir a busca na base real**

```bash
for q in "100410/2018" "28.636.579/0001-00" "Construtora" "secretaria municipal de saude" "dispensa de licitacao"; do
  curl -s "localhost:8080/v1/search?q=$(python3 -c "import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1]))" "$q")" | python3 -c "import json,sys;d=json.load(sys.stdin);print(sys.argv[1], d.get('total', len(d.get('hits', d.get('results', [])))))" "$q"
done
```

(com `make run-api` rodando; confira o nome do campo de total na resposta da busca em `presentation/http`). Compare com as contagens da Tarefa 5, Step 6.

- [ ] **Step 3: Gerar o arquivo do dump**

```bash
docker compose exec postgres pg_dump -U postgres -d diario -Fc --no-owner --no-privileges -f /tmp/diario.dump
docker compose cp postgres:/tmp/diario.dump ./diario.dump
ls -lh diario.dump
```

`diario.dump` fica fora do git (confira que `*.dump` está no `.gitignore`; se não estiver, acrescente e commite com `chore: ignora dumps locais`).
