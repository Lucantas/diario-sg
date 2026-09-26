# Cadastro da Receita Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Carregar todo mês o cadastro da Receita (empresa, estabelecimento, sócios) dos CNPJs citados nos Diários e mostrá-lo na página da empresa, no MCP e nos painéis.

**Architecture:** Um job do módulo da API (`cmd/receita`) lê os zips do WebDAV da Receita por `Range`, filtra pelos CNPJs de `entities`, guarda as linhas filtradas no bucket e troca, numa transação, as tabelas `rf_*` e as ligações da fonte. A leitura (`GetCompany`, `GetEntity`, painel) junta o cadastro pelo CNPJ.

**Tech Stack:** Go (`archive/zip`, `encoding/csv`, `net/http`), Postgres, React, Terraform.

**Spec:** `docs/superpowers/specs/2026-09-26-cadastro-da-receita-design.md`

## Global Constraints

- Sem comentários no código; conventional commits em português, sem link de sessão; commits como Lucas Dantas <lucas.lucantas38@gmail.com>.
- Migrations aplicadas não são editadas; a nova é `014_receita.sql`.
- Fonte: `receita_cnpj`; WebDAV `https://arquivos.receitafederal.gov.br/public.php/webdav/`, token do compartilhamento `YggdBLfdninEJX9` (variável `RECEITA_SHARE_TOKEN`, com esse padrão).
- Sócio só dentro da página da empresa (ADR 0006); o dump não muda.
- `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.
- Nada rodando ao final.

---

### Task 1: Modelo, ADR e migration

**Files:**
- Create: `docs/adr/0008-cadastro-da-receita.md`, `services/api/migrations/014_receita.sql`, `services/api/internal/core/domain/registry.go`, `services/api/internal/core/domain/registry_test.go`

**Interfaces:**
- Produces:
  - `domain.SourceReceita = "receita_cnpj"`, `domain.RecordEstablishment = "estabelecimento"`
  - `type RegistryCompany struct { Base, Name, LegalNature string; CapitalCents int64; Size string }`
  - `type RegistryEstablishment struct { CNPJ string; Headquarters bool; TradeName, Status string; StatusSince *time.Time; StatusReason string; OpenedAt *time.Time; MainActivity Activity; OtherActivities []Activity; Street, Number, Complement, District, ZIP, City, UF string }`
  - `type Activity struct { Code, Description string }`
  - `type RegistryPartner struct { Base string; Kind PartnerKind; Name, Document, Role string; Since *time.Time }` com `PartnerKind` `PartnerCompany|PartnerPerson|PartnerForeign` (códigos 1, 2, 3)
  - `type CompanyRegistry struct { Month time.Time; Company RegistryCompany; Establishment RegistryEstablishment; Partners []RegistryPartner }`
  - `type RegistryCodes struct { Activities, Cities, Natures, Roles, Reasons map[string]string }`
  - `ParseCompanyRow(f []string, c RegistryCodes) (RegistryCompany, error)`, `ParseEstablishmentRow(f []string, c RegistryCodes) (RegistryEstablishment, error)`, `ParsePartnerRow(f []string, c RegistryCodes) (RegistryPartner, error)`
  - `CNPJBase(cnpj string) string`

- [x] Testes com linhas reais de cada arquivo (nomes trocados): empresa (`"07396865";"EMPRESA X LTDA";"2062";"49";"1500000,00";"03";""` → capital 150000000, porte "Demais" para 05, "Micro empresa" 01, "Pequeno porte" 03); estabelecimento (situação 02 → "Ativa", 08 → "Baixada", 04 → "Inapta", 03 → "Suspensa", 01 → "Nula"; data `0` ou `00000000` → nil; CNAEs secundários separados por vírgula; município pelo código); sócio (tipo 2 → pessoa física, documento `***123456**` mantido; qualificação pelo código). Linha com número de campos errado dá erro.
- [x] Implementar `registry.go`.
- [x] Migration 014:
  ```sql
  CREATE TABLE rf_companies (cnpj_base text PRIMARY KEY, name text NOT NULL, legal_nature text NOT NULL, capital_cents bigint NOT NULL, size text NOT NULL, reference_month date NOT NULL);
  CREATE TABLE rf_establishments (cnpj text PRIMARY KEY, cnpj_base text NOT NULL, headquarters boolean NOT NULL, trade_name text NOT NULL, status text NOT NULL, status_since date, status_reason text NOT NULL, opened_at date, main_activity_code text NOT NULL, main_activity text NOT NULL, other_activities jsonb NOT NULL, street text NOT NULL, number text NOT NULL, complement text NOT NULL, district text NOT NULL, zip text NOT NULL, city text NOT NULL, uf text NOT NULL, reference_month date NOT NULL);
  CREATE INDEX rf_establishments_base_idx ON rf_establishments (cnpj_base);
  CREATE TABLE rf_partners (cnpj_base text NOT NULL, kind smallint NOT NULL CHECK (kind IN (1, 2, 3)), name text NOT NULL, document text NOT NULL, role text NOT NULL, since date, reference_month date NOT NULL);
  CREATE INDEX rf_partners_base_idx ON rf_partners (cnpj_base);
  ```
- [x] ADR 0008: carga filtrada no job da API (exceção ao coletor sem banco do ADR 0005), só o mês mais recente, arquivo bruto só com as linhas filtradas, sócio sob ADR 0006.
- [x] `make lint test`; commit `feat(api): modelo do cadastro da Receita`.

### Task 2: Leitura dos zips da Receita

**Files:**
- Create: `services/api/internal/adapters/receita/webdav.go`, `rangereader.go`, `rows.go` e testes.

**Interfaces:**
- Consumes: `domain.Parse*Row`, `domain.RegistryCodes`, `domain.CNPJBase`
- Produces:
  - `receita.New(baseURL, token string, client *http.Client) *Source`
  - `(*Source).LatestMonth(ctx) (string, error)` (maior diretório `AAAA-MM` do PROPFIND)
  - `(*Source).Files(ctx, month) ([]string, error)`
  - `(*Source).Rows(ctx, month, file string, each func(fields []string, raw string) error) (sha256 string, err error)`: abre o zip por `Range`, decodifica latin-1, chama `each` por linha, devolve o SHA-256 do zip lido
  - `(*Source).Codes(ctx, month) (domain.RegistryCodes, error)`
- [x] `rangeReaderAt`: `ReadAt` com bloco de 16 MB em cache, 3 tentativas por pedido; teste contra `httptest.Server` que serve um zip gerado no teste e responde 206 só a `Range`.
- [x] `Rows` com zip gerado (CSV `;`, latin-1 com `Ç`), confere campos e texto em UTF-8.
- [x] `LatestMonth` com resposta PROPFIND gravada no teste.
- [x] `make lint test`; commit `feat(api): leitura dos dados abertos do CNPJ`.

### Task 3: Carga no banco e job

**Files:**
- Create: `services/api/internal/adapters/postgres/registry.go`, `services/api/internal/core/usecase/load_registry.go` (+ teste com fakes), `services/api/cmd/receita/main.go`, `services/api/internal/integration/registry_test.go`
- Modify: `internal/core/ports/ports.go`, `internal/config/config.go` (`RoleReceita`, `RECEITA_SHARE_TOKEN`, `RAW_BUCKET` opcional = `GAZETTE_BUCKET`), `Makefile` (`receita: ## Carrega o cadastro da Receita dos CNPJs citados: make receita [MONTH=AAAA-MM]`)

**Interfaces:**
- Produces:
  - `ports.RegistrySource` (métodos do `*receita.Source`)
  - `ports.RegistryRepository { CitedCNPJs(ctx) ([]string, error); Replace(ctx, month time.Time, r domain.RegistryLoad) error }`
  - `ports.RegistryReader { RegistryByCNPJ(ctx, cnpj string) (*domain.CompanyRegistry, error); NamesByCNPJ(ctx, cnpjs []string) (map[string]string, error) }`
  - `domain.RegistryLoad { Companies []RegistryCompany; Establishments []RegistryEstablishment; Partners []RegistryPartner }`
  - `usecase.NewLoadRegistry(src, repo, runs ports.FetchRunRepository, raw ports.ObjectWriter, now func() time.Time).Execute(ctx, month string) (domain.FetchRun, error)`
- [x] Teste do caso de uso: filtra empresa e sócios pelo básico e estabelecimento pelo exato; grava `raw/receita_cnpj/AAAA/MM/01/<arquivo>.csv.gz`; `fetch_runs` com encontrados, gravados, pulados; erro num arquivo não chama `Replace` e registra o erro.
- [x] `Replace` numa transação: apaga `rf_*` e `entity_links` com `source = receita_cnpj`, insere, liga cada estabelecimento ao CNPJ (`exata`, evidência = razão social).
- [x] Integração: carga duas vezes (troca o mês), ligações criadas, `RegistryByCNPJ` com sócios, `NamesByCNPJ`.
- [x] `cmd/receita` como `cmd/dump`; mês padrão = `LatestMonth`.
- [x] `make lint test test-integration`; commit `feat(api): carga mensal do cadastro da Receita`.

### Task 4: API, MCP e painéis

**Files:**
- Modify: `usecase/queries.go` (`GetCompany`), `usecase/get_entity.go`, `usecase/supplier_panel.go`, `domain/entity.go` (`Registry *CompanyRegistry` em `CompanyReport` e `EntityReport`), `domain/supplier_panel.go` (`Name` em `SupplierRow`), `presentation/http/router.go` e `panels.go`, `presentation/mcp/` (ferramenta `entidade`), `cmd/api/main.go`, testes e integração.
- [x] `registry` no JSON de `/v1/entities/cnpj/{cnpj}` (nulo sem cadastro), com `month`, empresa, estabelecimento e `partners`.
- [x] `entidade` do MCP traz `cadastro_receita` quando o tipo é CNPJ; descrição da ferramenta cita a fonte e o mês.
- [x] `name` em cada fornecedor do painel.
- [x] `make lint test test-integration`; commit `feat(api): cadastro da Receita na empresa, no MCP e nos painéis`.

### Task 5: Página

**Files:**
- Modify: `apps/web/src/api.ts`, `CompanyPage.tsx`, `PanelsPage.tsx`, `styles.css`; Create: `apps/web/src/registry.ts` e `registry.test.ts`
- [x] `registry.ts`: `formatAddress`, `isActive(status)`, `formatPartnerSince`; testes.
- [x] Seção "Cadastro na Receita" (razão social no título, situação em destaque quando não ativa, sócios, "Dados da Receita de <mês>", aviso quando não há cadastro); nome no cartão do painel.
- [x] `npm run typecheck && npm test`; Playwright da empresa e de `/paineis` em 1280 e 390 px sem rolagem horizontal; commit `feat(web): cadastro da Receita na página da empresa`.

### Task 6: Nuvem, carga real e docs

- [x] Terraform: job `receita` e agendamento mensal (dia 20, 06:00), como o `dump`; `make tf-fmt`.
- [x] Carga real local de 2026-09; conferir a razão social de 20 empresas contra o nome citado no Diário; medir cobertura (CNPJs com cadastro / citados com dígito verificador válido).
- [x] Roadmap (Entrega 3 parte 1 ✅, com números; pendência de nuvem), `README.md` (fonte e comando).
- [x] Commit `docs(roadmap): cadastro da Receita`.
