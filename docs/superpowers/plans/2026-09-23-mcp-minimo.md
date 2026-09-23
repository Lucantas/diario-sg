# MCP mínimo (etapa A): plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** servidor MCP remoto, só leitura, em `POST /mcp` da API, com chave por usuário gerada no site e as ferramentas `buscar_atos`, `ler_ato`, `entidade` e `fontes`.

**Architecture:** o pacote `presentation/mcp` registra as ferramentas no SDK oficial (`Stateless`, `JSONResponse`) e chama os casos de uso que a API já usa. Um middleware autentica a chave (`usecase.APIKeys`), aplica o limite por chave e guarda a chave no contexto para o registro de uso. O roteador HTTP monta o handler em `/mcp` e expõe a emissão e a revogação da chave.

**Tech Stack:** Go 1.25, `github.com/modelcontextprotocol/go-sdk` v1.8.0, PostgreSQL 16, React 18, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-23-mcp-minimo-design.md`

## Global Constraints

- Zero comentários novos no código; migrations aplicadas não mudam.
- Chave: `dsg_` + 32 bytes aleatórios em base64url sem padding; o banco guarda o SHA-256 em hex e os 8 primeiros caracteres depois de `dsg_`.
- Limites: criação 3/h por cliente e 50/h por instância; chamadas MCP 60/min por chave e 600/min por instância.
- Até 20 atos por chamada; `limite` padrão 10.
- Conventional commits em português, sem link de sessão de agente.
- Antes de concluir: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.

---

### Task 1: Go 1.25 e SDK

**Files:** `services/api/go.mod`, `services/api/go.sum`, `go.work`, `services/api/Dockerfile`

- [ ] `go get github.com/modelcontextprotocol/go-sdk@v1.8.0` em `services/api`; `go` do módulo e do `go.work` em `1.25`.
- [ ] `FROM golang:1.25-alpine` na imagem da API.
- [ ] `make lint test` verde.
- [ ] Commit `chore(api): Go 1.25 e SDK de MCP`.

### Task 2: limitador em pacote próprio

**Files:** criar `internal/presentation/ratelimit/{ratelimit.go,ratelimit_test.go}`; remover `internal/presentation/http/ratelimit*.go`; ajustar `router.go`, `reports.go`.

**Produces:** `ratelimit.New(perClient, total int, window time.Duration, now func() time.Time) *Limiter`, `(*Limiter).Allow(key string) bool`. `clientKey` fica no pacote `http` (`client.go`).

- [ ] Mover os testes atuais para o pacote novo e rodar (falham sem o pacote).
- [ ] Mover o código; `make lint test` verde.
- [ ] Commit `refactor(api): limitador de requisições em pacote próprio`.

### Task 3: chaves no domínio, no banco e no caso de uso

**Files:** `migrations/007_api_keys.sql`; `domain/apikey.go` (+ teste); `domain/errors.go` (`ErrUnauthorized`); `ports/ports.go`; `usecase/api_keys.go` (+ teste com repositório em memória); `adapters/postgres/api_keys.go`.

**Produces:**
- `domain.APIKey{ID, Prefix string; CreatedAt time.Time}`; `domain.NewAPIKeySecret() (string, error)`; `domain.HashAPIKey(secret string) string`; `domain.APIKeyPrefix(secret string) string`.
- `ports.APIKeyRepository`: `Create(ctx, hash, prefix string) (domain.APIKey, error)`, `FindActive(ctx, hash string) (domain.APIKey, error)` (`ErrNotFound` se não existe ou foi revogada), `Revoke(ctx, hash string) error`, `RecordUse(ctx, keyID, tool string) error`.
- `usecase.APIKeys`: `Issue(ctx) (secret string, key domain.APIKey, err error)`, `Authenticate(ctx, secret) (domain.APIKey, error)` (`ErrUnauthorized`), `Revoke(ctx, secret) error` (`ErrUnauthorized`), `RecordUse(ctx, keyID, tool) error`.

- [ ] Testes de domínio e de caso de uso primeiro; rodar e ver falhar.
- [ ] Implementar; `make lint test` verde.
- [ ] Commit `feat(api): chaves de acesso para o MCP`.

### Task 4: cobertura e leitura de um ato

**Files:** `domain/coverage.go`; `ports/ports.go` (`CoverageReader`); `usecase/read_act.go`, `usecase/coverage.go` (+ testes); `adapters/postgres/gazettes.go` (`Coverage`).

**Produces:**
- `domain.Coverage{First, Last, LastIndexedAt time.Time; Gazettes, Acts int}`.
- `ports.CoverageReader`: `Coverage(ctx) (domain.Coverage, error)`.
- `usecase.ReadAct.Execute(ctx, gazetteID string, position int) (domain.Gazette, domain.Act, error)`.
- `usecase.SourceCoverage.Execute(ctx) (domain.Coverage, error)`.

- [ ] Testes de `ReadAct` (acha pela posição; posição inexistente é `ErrNotFound`) e de `SourceCoverage`.
- [ ] Implementar; commit `feat(api): cobertura da base e leitura de um ato`.

### Task 5: servidor MCP

**Files:** `internal/presentation/mcp/{server.go,tools.go,dto.go,citation.go,auth.go}` (+ `citation_test.go`, `dto_test.go`).

**Consumes:** `SearchActs`, `ReadAct`, `GetCompany`, `SourceCoverage`, `APIKeys`, `ratelimit.Limiter`.

**Produces:** `mcp.Deps{Search, Read, Company, Coverage, Keys, PublicWebURL, Log}`; `mcp.NewHandler(d Deps) http.Handler`.

- [ ] Teste da citação: mesma saída de `formatCitation` do site para um ato com páginas 3–4 e para um sem página.
- [ ] Teste de `fontes` (URL oficial e cópia com `#page=N`) e de `reaisToCents` (`1500.5` → `150050`, negativo é erro).
- [ ] Implementar ferramentas com `mcp.AddTool` e `ReadOnlyHint`; middleware: `Authorization: Bearer` → `Authenticate` (401 com `WWW-Authenticate: Bearer`), `Allow(key.ID)` (429), chave no contexto; cada ferramenta chama `RecordUse` e loga `key_prefix` e ferramenta.
- [ ] Commit `feat(api): servidor MCP com busca, leitura, CNPJ e fontes`.

### Task 6: rotas e integração

**Files:** `presentation/http/{router.go,mcp_keys.go,dto.go}`; `cmd/api/main.go`; `internal/integration/mcp_test.go`.

- [ ] Teste de integração com o cliente do SDK (`mcp.NewClient`, `StreamableClientTransport` com `HTTPClient` que põe o cabeçalho): 401 sem chave; `POST /v1/mcp/keys` → 201 com `key`; quatro ferramentas listadas; `buscar_atos`, `ler_ato` do primeiro resultado, `entidade`, `fontes`; `api_key_usage` com uma linha por ferramenta; `DELETE /v1/mcp/keys` → 204 e depois 401; quarta criação do mesmo cliente → 429; campo isca preenchido → 201 sem gravar.
- [ ] `POST /v1/mcp/keys`, `DELETE /v1/mcp/keys`, `mux.Handle("/mcp", a.MCP)`; `ErrUnauthorized` → 401 em `writeError`.
- [ ] `make test-integration` verde; commit `feat(api): rotas do MCP e das chaves`.

### Task 7: página do MCP no site

**Files:** `apps/web/src/{McpPage.tsx,mcp.ts,mcp.test.ts,api.ts,App.tsx,SearchPage.tsx,styles.css}`.

- [ ] Teste de `mcpSnippets(url, key)`: Claude Code (`claude mcp add --transport http diario-sg <url> --header "Authorization: Bearer <key>"`), Claude Desktop (`npx mcp-remote <url> --header …`) e Cursor (JSON com `url` e `headers`).
- [ ] Página `/mcp`; link no rodapé da busca; `npm run typecheck && npm test`.
- [ ] Commit `feat(web): página para gerar a chave do MCP`.

### Task 8: documentação e verificação

**Files:** `README.md`, `docs/roadmap.md`, `docs/plano-fontes-publicas.md`, `docs/decisoes-de-codigo.md`.

- [ ] README: rotas novas, seção "Servidor MCP" com URL, chave e exemplo; roadmap e plano: etapa A entregue e o que ficou (OAuth, CLI de chaves).
- [ ] Verificação completa e teste manual com a API local e um cliente MCP.
- [ ] Commit `docs(mcp): servidor MCP no README, no roadmap e no plano`.
