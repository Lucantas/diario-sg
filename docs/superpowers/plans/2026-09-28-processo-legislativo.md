# Processo legislativo (SICAM) e tema ambiental — plano

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Trazer os processos do SICAM (projeto, tramitação, pareceres) para a
base e responder "como andam as leis de um tema", começando pelo ambiental.

**Architecture:** Mesmo molde das normas do SIAPEGOV: adaptador HTTP →
parser no domínio → caso de uso de carga com `fetch_runs` e bruto no bucket →
repositório Postgres; leitura por caso de uso, HTTP, MCP e site. Fase, dias
sem movimentação, ligação com a lei e tema são calculados na leitura.

**Tech Stack:** Go (`services/api`), Postgres (`portuguese_unaccent`),
`golang.org/x/net/html` (já usado pelo `tableRows` das normas), SDK MCP em Go,
React + Vite (`apps/web`), Terraform (`infra/stack`).

**Spec:** `docs/superpowers/specs/2026-09-28-processo-legislativo-design.md`

## Global Constraints

- Nada de IA nos dados: fase, tema e ligação são regras escritas, e a
  ferramenta devolve a regra.
- 1 pedido por segundo ao SICAM, `User-Agent` `diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)`, só `/areapublica/` e `sitemap*.xml`.
- Não usar `api.sicam.app`.
- Sem comentários no código (AGENTS.md); mensagens e nomes de campo públicos em português.
- Conventional commits em português, sem link de sessão.
- Antes de dizer que terminou: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck`.

---

## File Structure

| Arquivo | Responsabilidade |
| --- | --- |
| `services/api/internal/core/domain/bill.go` | tipos `Bill`, `BillEvent`, `BillOpinion`, `BillKey`, tipos normativos |
| `services/api/internal/core/domain/bill_page.go` | `ParseBillPage`, `ParseProcessSitemap`, datas por extenso |
| `services/api/internal/core/domain/bill_phase.go` | `BillPhase`, `PhaseOf`, `DaysIdle` |
| `services/api/internal/core/domain/bill_reference.go` | `BillReference` (autor da norma → documento do projeto) |
| `services/api/internal/core/domain/theme.go` | tema `meio_ambiente`: termos, comissão, órgãos, regra em texto |
| `services/api/internal/core/domain/testdata/sicam/` | páginas reais gravadas (só o `<main>`) e sitemap reduzido |
| `services/api/migrations/030_bills.sql` | `bills`, `bill_events`, `bill_opinions` |
| `services/api/internal/adapters/postgres/bills.go` | gravação em lote, fila diária, consultas |
| `services/api/internal/adapters/sicam/source.go` | sitemap e página do processo, com pausa |
| `services/api/internal/core/usecase/load_bills.go` | carga completa e diária em lotes |
| `services/api/internal/core/usecase/find_bills.go` | lista e processo, com fase, lei e tema |
| `services/api/cmd/sicam/main.go` | job `sicam` |
| `services/api/internal/presentation/http/bills.go` | `GET /v1/bills`, `GET /v1/bills/{key}` |
| `services/api/internal/presentation/mcp/bills.go` | ferramenta `proposicoes` |
| `apps/web/src/BillsPage.tsx`, `BillPage.tsx`, `bills.ts` | página `/proposicoes` |

---

### Task 1: Parser da página do processo e do sitemap

**Files:**
- Create: `services/api/internal/core/domain/bill.go`, `bill_page.go`, `bill_page_test.go`
- Create: `services/api/internal/core/domain/testdata/sicam/{5564-2025,3865-2019,100-2025,inexistente}.html`, `sitemap-processos.xml`

**Interfaces:**
- Produces:
  ```go
  const SourceBills = "sicam_processos"
  type BillKey struct{ Number, Year int }
  func (k BillKey) String() string            // "5564/2025"
  func (k BillKey) Slug() string              // "5564-2025"
  func ParseBillKey(s string) (BillKey, error) // aceita 5564/2025, 5564-2025, 5564_2025
  type Bill struct {
      Key BillKey; Kind, DocLabel string; DocNumber, DocYear int
      Summary, Authors string; PresentedOn *time.Time
      Status, CurrentBody, LastMovement string; SourceUpdatedAt *time.Time
      URL string; FetchedAt time.Time
      Events []BillEvent; Opinions []BillOpinion
  }
  type BillEvent struct{ Position int; At time.Time; Label, Text, Sector string }
  type BillOpinion struct{ Position int; Result string; On *time.Time; Committee, Rapporteur string }
  var ErrBillNotFound = errors.New("processo não encontrado")
  func ParseBillPage(page []byte, base string) (Bill, error)
  func ParseProcessSitemap(xml []byte) ([]BillKey, error)
  func BillPageMain(page []byte) []byte  // só o <main>, para o bruto
  ```

- [ ] **Step 1:** Gravar as fixtures reais (só `<main>…</main>`) com `curl` a partir de `https://sg.processolegislativo.com.br/areapublica/processo/{slug}` e um sitemap com 3 `<url>` de processo e a de listagem.
- [ ] **Step 2: Testes que falham** — `TestParseBillPageMessageSentToExecutive` (5564-2025: tipo `MENSAGEM`, `DocLabel` `MENSAGEM Nº 032/2025`, doc 32/2025, situação `Ativo`, autor `PREFEITURA MUNICIPAL DE SÃO GONÇALO`, apresentação 18/12/2025, 22 eventos, primeiro evento 06/01/2026 11:27 America/Sao_Paulo com texto "Enviado para PREFEITURA MUNICIPAL DE SÃO GONÇALO - Ofício nº. 462/2025 em 29/12/2025", evento com setor "Setor de Expediente", 3 pareceres com a CJR relatada por NELSINHO RUAS); `TestParseBillPageArchivedAtEndOfTerm` (3865-2019: `PROJETO DE LEI Nº 270/2019`, `Arquivado`, 9 eventos, 1 parecer); `TestParseBillPageIndication` (100-2025); `TestParseBillPageNotFound` (`errors.Is(err, ErrBillNotFound)`); `TestParseProcessSitemap` (só as 3 chaves, ignora `/areapublica/processos`); `TestParseBillKey`.
- [ ] **Step 3:** `go test ./internal/core/domain -run 'Bill|Sitemap'` → FAIL.
- [ ] **Step 4: Implementar** com `golang.org/x/net/html`: classes `hero-card`, `hero-title` (spans), `ementa-box`, `Autor(es):`/`Apresentação:` pelo `<strong>`, `summary-box` (label → value), `timeline-node` (`timeline-date`, badge, `timeline-title`, `Setor:`), seção "Pareceres das Comissões". Mês por extenso em tabela; fuso `America/Sao_Paulo`.
- [ ] **Step 5:** testes → PASS; `gofmt`/`go vet`.
- [ ] **Step 6: Commit** `feat(sicam): parser da página do processo e do sitemap`

### Task 2: Fase, dias sem movimentação e ligação com a lei

**Files:**
- Create: `domain/bill_phase.go`, `bill_phase_test.go`, `bill_reference.go`, `bill_reference_test.go`
- Modify: `domain/bill.go` (tipos normativos)

**Interfaces:**
- Produces:
  ```go
  type BillPhase string // apresentado, em_comissao, em_votacao, aprovado, rejeitado, retirado, enviado_ao_executivo, arquivado
  var BillPhasesInOrder []BillPhase
  func ParseBillPhase(s string) (BillPhase, error)
  func PhaseOf(b Bill) BillPhase
  func DaysIdle(b Bill, today time.Time) int
  var NormativeBillKinds []string // PROJETO DE LEI, PROJETO DE LEI COMPLEMENTAR, PROJETO DE LEI SUBSTITUTIVO, PROJETO DE RESOLUÇÃO, PROJETO DE EMENDA À LEI ORGÂNICA, MENSAGEM, EMENDA
  func IsNormativeKind(kind string) bool
  type BillDocRef struct{ Kind string; Number, Year int }
  func BillReference(author string) (BillDocRef, bool)
  ```
- [ ] **Step 1: Testes que falham** — uma tramitação real por fase (textos copiados das fixtures e da amostra: "Processo Arquivado - Término de mandato", "Enviado para PREFEITURA…", "Aprovado - Votação única por 22 votos…", "Encaminhado ao setor Para Votação", "Recebido na Comissão…", só "Entrada no Protocolo Geral"); `DaysIdle` com o último evento; `BillReference` com autores reais: "PROJETO DE LEI Nº 0133/2019 VEREADOR ´PR…" → PL 133/2019; "PROJETO D LEI N 118/2019 VEREADOR BRUNO" → PL 118/2019; "PROJETO DE LEI 0254/2018 VEREADOR PAULO" → PL 254/2018; "VEREADOR … PROJETO DE LEI 110/22" → PL 110/2022; "VER. DINEY MARINS" → sem ligação; "PROJETO DE LEI COMPLEMENTAR Nº 3/2021" → PLC.
- [ ] **Step 2:** FAIL. **Step 3:** implementar (regras em ordem, texto sem acento e em minúsculas). **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(sicam): fase do processo e ligação do projeto com a lei`

### Task 3: Tema ambiental

**Files:**
- Create: `domain/theme.go`, `theme_test.go`

**Interfaces:**
- Produces:
  ```go
  type Theme struct {
      Slug, Name string
      Terms []string          // raízes sem acento
      Committee string        // "MEIO AMBIENTE"
      Organs []string         // SEMMA, SEMA, SEMMADU, COMMADS, PROMEA
      PartialOrgan string     // SEMMATRAN
      PartialOrganExcludes []string
      Rule string             // a regra em texto, devolvida pela ferramenta
  }
  func ParseTheme(slug string) (Theme, error) // "meio_ambiente"; vazio → Theme{} sem erro
  func (t Theme) TermsRegex() string          // POSIX para ~* em texto sem acento
  func (t Theme) MatchesText(s string) bool
  ```
- [ ] **Step 1: Testes que falham** com ementas reais: entram 1601/2025 (infrações ambientais), 1606/2025 (arborização), 1590/2025 (resíduos sólidos), 1131/2020 (proteção aos animais), 1618/2026 (ruídos de motocicleta), 1035/2019 (canudos de plástico); não entram 1085/2020 ("ambiente escolar"), 845/2018 ("ambiente laborativo"), 853/2018 ("produtos de origem animal"), 497/2025 (crédito suplementar da Secretaria de Conservação).
- [ ] **Step 2:** FAIL. **Step 3:** implementar. **Step 4:** PASS; conferir na base local a contagem de leis que casam antes/depois de ajustar termos.
- [ ] **Step 5: Commit** `feat(tema): regra do tema meio ambiente`

### Task 4: Tabelas e repositório

**Files:**
- Create: `services/api/migrations/030_bills.sql`, `adapters/postgres/bills.go`, `internal/integration/bills_test.go`
- Modify: `core/ports/ports.go`

**Interfaces:**
- Produces:
  ```go
  type BillRepository interface {
      Ready(ctx) error
      KnownBills(ctx) (map[BillKey]bool, error)
      StaleOpenBills(ctx, kinds []string, limit int) ([]BillKey, error) // não arquivados, lidos há mais tempo primeiro
      SaveBills(ctx, bills []Bill) error // upsert + troca eventos e pareceres, uma transação
  }
  type BillFilter struct {
      Text, Author string; Kinds []string; AllKinds bool
      Phase BillPhase; Status string; MinIdleDays int
      Theme Theme; From, To time.Time; Limit, Offset int
  }
  type BillReader interface {
      BillByKey(ctx, key BillKey) (Bill, bool, error)
      BillsByDoc(ctx, ref BillDocRef) ([]Bill, error)
      SearchBills(ctx, f BillFilter, today time.Time) ([]Bill, int, map[BillPhase]int, error)
      NormsForBill(ctx, b Bill) ([]Norm, error)
      BillForNorm(ctx, n Norm) (*Bill, error)
  }
  ```
  `SearchBills` carrega os eventos (para a fase) dos processos filtrados por texto/tipo/autor/tema/data, calcula fase e dias em Go, filtra por fase e `MinIdleDays`, conta por fase, ordena por última movimentação mais antiga quando `MinIdleDays>0` e por apresentação mais recente nos demais, e pagina.
- [ ] **Step 1:** migration 030 (PK `(process_number, process_year)`; índices de texto em `summary || ' ' || authors` e por `kind`; `bill_events`/`bill_opinions` com FK `ON DELETE CASCADE`).
- [ ] **Step 2: Teste de integração que falha** — grava dois lotes (o segundo troca os eventos do mesmo processo), confere `KnownBills`, `StaleOpenBills` (não traz arquivado nem indicação), `SearchBills` por texto, tema (ementa e comissão), fase e `MinIdleDays`, e `NormsForBill` com uma norma cujo autor cita o projeto.
- [ ] **Step 3:** `make test-integration` → FAIL. **Step 4:** implementar. **Step 5:** PASS.
- [ ] **Step 6: Commit** `feat(sicam): tabelas e repositório dos processos`

### Task 5: Coleta (adaptador, caso de uso, job)

**Files:**
- Create: `adapters/sicam/source.go`, `source_test.go` (httptest), `usecase/load_bills.go`, `load_bills_test.go` (fakes), `cmd/sicam/main.go`
- Modify: `config/config.go` (papel `RoleSICAM`, `SICAM_SITE_URL` padrão `https://sg.processolegislativo.com.br/`), `Makefile` (`sicam: ## …`), `ports.go` (`BillSource`)

**Interfaces:**
- Produces:
  ```go
  type BillSource interface {
      BaseURL() string
      ProcessKeys(ctx) ([]BillKey, error)  // índice + sitemap-processos-*.xml
      ProcessPage(ctx, key BillKey) ([]byte, error)
  }
  func NewLoadBills(src BillSource, repo BillRepository, runs FetchRunRepository, raw ObjectWriter, now func() time.Time) *LoadBills
  func (uc *LoadBills) Execute(ctx, full bool, max int) (FetchRun, error)
  ```
- [ ] **Step 1: Testes que falham** — adaptador: pausa entre pedidos, `User-Agent`, lê o índice e os dois sitemaps; caso de uso: modo diário pega novos + parados até `max`, grava em lotes de 500, `ErrBillNotFound` conta em `skipped`, erro de parser em `failed` sem parar, erro de rede para a coleta mas o lote já gravado fica, bruto `raw/sicam_processos/AAAA/MM/DD/<run>-<lote>.jsonl` com manifesto.
- [ ] **Step 2:** FAIL. **Step 3:** implementar. **Step 4:** PASS.
- [ ] **Step 5:** rodar `make migrate` e `make sicam MAX=30` na base local; conferir as linhas.
- [ ] **Step 6: Commit** `feat(sicam): job de coleta dos processos da Câmara`
- [ ] **Step 7:** disparar a carga completa em segundo plano (`make sicam FULL=1`), ≈14 h.

### Task 6: Leitura — `/v1/bills` e ferramenta `proposicoes`

**Files:**
- Create: `usecase/find_bills.go`, `find_bills_test.go`, `presentation/http/bills.go`, `presentation/mcp/bills.go`, `internal/integration/bills_http_test.go`
- Modify: `cmd/api/main.go`, `http/router.go`, `mcp/server.go`, `mcp/tools.go`, `mcp/norms.go` (projeto de origem), `usecase/find_norms.go`

**Interfaces:**
- Consumes: `BillReader`, `ParseTheme`, `PhaseOf`, `DaysIdle`, `BillReference`.
- Produces: `FindBills.List(ctx, BillQuery) (BillPage, error)`, `FindBills.One(ctx, key string) (BillDetail, error)`, `FindBills.ByDoc(ctx, kind, number string)`.
  Ferramenta `proposicoes` (entrada: `processo`, `tipo`, `numero`, `texto`, `autor`, `fase`, `situacao`, `parado_ha_dias`, `tema`, `de`, `ate`, `deslocamento`, `limite`); saída com `total`, `por_fase`, `regra_tema`, e por item `processo`, `documento`, `tipo`, `ementa`, `autores`, `apresentacao`, `situacao`, `fase`, `dias_sem_movimentacao`, `orgao_atual`, `ultima_movimentacao`, `link`, `leis`; com `processo`, também `tramitacao` e `pareceres`.
- [ ] **Step 1:** testes do caso de uso (validação de fase, tema, tipo, número) e de integração HTTP e MCP.
- [ ] **Step 2:** FAIL. **Step 3:** implementar. **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(mcp): ferramenta proposicoes e GET /v1/bills`

### Task 7: Tema em `norma`, `buscar_atos` e `agrupar`

**Files:**
- Modify: `domain/act.go` (`ActFilter.Theme`), `adapters/postgres/filter.go`, `filter_test.go`, `ports.go`/`norms.go` (`SearchNorms` com tema), `mcp/tools.go`, `mcp/norms.go`, `mcp/group.go`, `http` da busca (`theme=`), `usecase/find_norms.go`
- [ ] **Step 1:** teste de `filterSQL` com tema (órgãos, SEMMATRAN sem trânsito, título) e integração: um ato da SEMMATRAN de licença entra, um do CORIM não; `norma` com `tema` e sem texto lista as normas do tema.
- [ ] **Step 2:** FAIL. **Step 3:** implementar. **Step 4:** PASS; conferir contagens na base local.
- [ ] **Step 5: Commit** `feat(tema): filtro de tema em buscar_atos, agrupar e norma`

### Task 8: Página `/proposicoes`

**Files:**
- Create: `apps/web/src/bills.ts`, `bills.test.ts`, `BillsPage.tsx`, `BillPage.tsx`
- Modify: `apps/web/src/App.tsx` (rotas e menu), `api.ts` (tipos e chamadas), `styles.css`
- [ ] **Step 1:** teste (vitest) da montagem da URL de filtros e dos rótulos de fase. **Step 2:** FAIL. **Step 3:** implementar lista (filtros texto, tema, fase, parado há, tipo) e página do processo (linha do tempo, pareceres, lei, busca no Diário). **Step 4:** `npm test`, `npm run typecheck`; conferir no navegador (Playwright headless).
- [ ] **Step 5: Commit** `feat(web): página de proposições da Câmara`

### Task 9: Fontes, infra e documentação

**Files:**
- Modify: `domain/source.go`/cobertura (fonte `sicam_processos` em `fontes`), `mcp/fontes.md`, `docs/fontes/README.md`, `docs/plano-fontes-publicas.md`, `README.md` (API e jobs), `infra/stack/main.tf` e `variables.tf` (job `sicam`, conta de serviço, agendamento diário 05:00), `docs/roadmap.md`
- [ ] **Step 1:** teste de integração de `fontes` com uma `fetch_run` de `sicam_processos`. **Step 2:** FAIL → implementar → PASS.
- [ ] **Step 3:** `terraform fmt` e `terraform validate` em `infra/stack`.
- [ ] **Step 4:** depois da carga completa: números reais no roadmap (processos por tipo e fase, projetos ambientais parados) e caixas da Entrega 6 marcadas.
- [ ] **Step 5: Commit** `docs(roadmap): processo legislativo entregue` (e `feat(infra): job sicam`).
