# Exportação da busca — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `GET /v1/acts/export` em CSV (Excel pt-BR) e JSON, com até 10.000 atos por busca, e links de download no front.

**Architecture:** `usecase.ExportActs` troca o limite do filtro por `domain.ExportLimit` e repassa a `ActRepository.Export`, que percorre as linhas e chama `yield`. O handler escreve cabeçalhos na primeira linha e faz streaming por um `exportWriter` (CSV ou JSON).

**Tech Stack:** Go 1.23 (`encoding/csv`, `encoding/json`), PostgreSQL 16, React 18, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-23-exportacao-da-busca-design.md`

## Global Constraints

- Zero comentários novos no código.
- Sem migration.
- CSV: separador `;`, UTF-8 com BOM, CRLF, decimal com vírgula, célula de texto que começa com `=`, `+`, `-`, `@`, `\t` ou `\r` ganha `'` na frente.
- Teto `domain.ExportLimit = 10000`.
- Conventional commits em português, sem link de sessão de agente.
- Antes de concluir: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.

---

### Task 1: Caso de uso e repositório

**Files:**
- Modify: `services/api/internal/core/domain/act.go` (`ExportLimit`)
- Modify: `services/api/internal/core/ports/ports.go` (`ActRepository.Export`)
- Create: `services/api/internal/core/usecase/export_acts.go`, `export_acts_test.go`
- Modify: `services/api/internal/adapters/postgres/acts.go` (`Export`, ordem compartilhada)
- Modify: `services/api/internal/adapters/postgres/search.go` (`orderClause`)

**Interfaces:**
- Produces: `domain.ExportLimit`; `ports.ActRepository.Export(ctx context.Context, f domain.ActFilter, yield func(h domain.ActHit, total int) error) error`; `usecase.NewExportActs(ports.ActRepository) *ExportActs`; `(*ExportActs).Execute(ctx, f domain.ActFilter, yield func(domain.ActHit, int) error) error`.

- [ ] **Step 1: Teste unitário que falha**

```go
type exportSpy struct {
	ports.ActRepository
	got domain.ActFilter
}

func (s *exportSpy) Export(_ context.Context, f domain.ActFilter, _ func(domain.ActHit, int) error) error {
	s.got = f
	return nil
}

func TestExportActsIgnoresPaginationAndUsesTheExportLimit(t *testing.T) {
	spy := &exportSpy{}

	err := NewExportActs(spy).Execute(context.Background(),
		domain.ActFilter{Query: "a OU b", Limit: 20, Offset: 40}, func(domain.ActHit, int) error { return nil })

	if err != nil || spy.got.Limit != domain.ExportLimit || spy.got.Offset != 0 || spy.got.Query != "a or b" {
		t.Fatalf("filtro inesperado: %+v %v", spy.got, err)
	}
}

func TestExportActsRejectsInvalidFilter(t *testing.T) {
	err := NewExportActs(&exportSpy{}).Execute(context.Background(),
		domain.ActFilter{Organ: "TOTAL"}, func(domain.ActHit, int) error { return nil })

	if !errors.Is(err, domain.ErrInvalidFilter) {
		t.Fatalf("esperava ErrInvalidFilter, veio %v", err)
	}
}
```

- [ ] **Step 2:** `go test ./internal/core/usecase/` → FAIL.
- [ ] **Step 3: Implementação**

```go
type ExportActs struct{ acts ports.ActRepository }

func NewExportActs(a ports.ActRepository) *ExportActs { return &ExportActs{acts: a} }

func (uc *ExportActs) Execute(ctx context.Context, f domain.ActFilter, yield func(domain.ActHit, int) error) error {
	if err := f.Normalize(); err != nil {
		return err
	}
	f.Limit, f.Offset = domain.ExportLimit, 0
	return uc.acts.Export(ctx, f, yield)
}
```

Postgres: `orderClause` com o `ORDER BY` de `Search` (usado pelos dois). `Export`:

```sql
SELECT a.id, a.gazette_id, a.type, a.title, a.body, a.position, a.organ, coalesce(a.page_start, 0), coalesce(a.page_end, 0),
       g.edition_number, g.published_at, g.is_extra, g.source_url, g.checksum,
       <cnpjsSubquery>, <valuesSubquery>, count(*) OVER ()
FROM acts a JOIN gazettes g ON g.id = a.gazette_id
CROSS JOIN websearch_to_tsquery('portuguese_unaccent', $1) q
<exactPhraseFor("$3")>
WHERE ($1 = '' OR <matchFor("$1", "$3")>) <filterSQL(f, 4)>
<orderClause>
LIMIT $2
```

com `yield(h, total)` por linha; erro do `yield` devolve o erro.
- [ ] **Step 4:** `go test ./...` → PASS.
- [ ] **Step 5:** commit `feat(api): caso de uso de exportação de atos`.

### Task 2: CSV e JSON na API

**Files:**
- Create: `services/api/internal/presentation/http/export.go`, `export_test.go`
- Modify: `services/api/internal/presentation/http/router.go` (`API.Export`, `API.PublicWebURL`, rota)
- Modify: `services/api/cmd/api/main.go`
- Create: `services/api/internal/integration/export_test.go`

**Interfaces:**
- Consumes: `usecase.ExportActs`.
- Produces: `csvCell(string) string`; `reaisBR(int64) string` (`120000` → `"1200,00"`); `archivedURL(base string, h domain.ActHit) string`; `csvRecord(base string, h domain.ActHit) []string`; `exportHeader []string`.

- [ ] **Step 1: Testes unitários que falham**

```go
func TestCSVCellNeutralizesFormulas(t *testing.T) {
	cases := map[string]string{"=1+1": "'=1+1", "+55": "'+55", "-x": "'-x", "@SUM": "'@SUM", "\tx": "'\tx", "PORTARIA": "PORTARIA", "": ""}
	for in, want := range cases {
		if got := csvCell(in); got != want {
			t.Errorf("csvCell(%q) = %q, esperava %q", in, got, want)
		}
	}
}

func TestReaisBR(t *testing.T) {
	cases := map[int64]string{0: "0,00", 5: "0,05", 120000: "1200,00", 123456789: "1234567,89"}
	for in, want := range cases {
		if got := reaisBR(in); got != want {
			t.Errorf("reaisBR(%d) = %q, esperava %q", in, got, want)
		}
	}
}

func TestCSVRecord(t *testing.T) {
	h := domain.ActHit{Act: domain.Act{GazetteID: "g1", Type: domain.ActContrato, Title: "EXTRATO", Body: "=corpo",
		PageStart: 3, PageEnd: 4, Organ: "FMS"}, EditionNumber: "1771", PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
		IsExtra: true, SourceURL: "https://do/x.pdf", Checksum: "abc", CNPJs: []string{"1", "2"}, ValuesCents: []int64{120000, 30000}}

	got := csvRecord("https://site", h)

	want := []string{"2026-09-18", "1771", "sim", "contrato", "FMS", domain.OrganName("FMS"), "EXTRATO", "3", "4",
		"1200,00 | 300,00", "1 | 2", "https://do/x.pdf#page=3", "https://site/api/v1/gazettes/g1/pdf#page=3", "abc", "'=corpo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("veio\n%q\nesperava\n%q", got, want)
	}
}

func TestCSVRecordWithoutPage(t *testing.T) {
	h := domain.ActHit{Act: domain.Act{GazetteID: "g1", Type: domain.ActOutro}, SourceURL: "https://do/x.pdf",
		PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)}

	got := csvRecord("https://site", h)

	if got[7] != "" || got[8] != "" || got[11] != "https://do/x.pdf" || got[12] != "https://site/api/v1/gazettes/g1/pdf" || got[2] != "não" {
		t.Errorf("veio %q", got)
	}
}
```

- [ ] **Step 2:** `go test ./internal/presentation/http/` → FAIL.
- [ ] **Step 3: Implementação** (`export.go`)

```go
var exportHeader = []string{"data", "edicao", "extra", "tipo", "orgao", "orgao_nome", "titulo", "pagina_inicio", "pagina_fim",
	"valores_reais", "cnpjs", "pdf_original", "pdf_arquivado", "sha256_pdf", "texto"}

func csvCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func reaisBR(cents int64) string {
	return fmt.Sprintf("%d,%02d", cents/100, cents%100)
}

func pageSuffix(h domain.ActHit) string {
	if h.PageStart <= 0 {
		return ""
	}
	return "#page=" + strconv.Itoa(h.PageStart)
}

func archivedURL(base string, h domain.ActHit) string {
	return strings.TrimRight(base, "/") + "/api/v1/gazettes/" + h.GazetteID + "/pdf" + pageSuffix(h)
}
```

`csvRecord` monta as 15 colunas (`pageText` devolve `""` para 0); `gazettePDF` e demais inalterados.

Handler:

```go
func (a *API) exportActs(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "json" {
		writeError(w, domain.ErrInvalidFilter, a.Log)
		return
	}
	f, ok := a.filterFromQuery(w, r)
	if !ok {
		return
	}
	out := newExportWriter(format, w, a.PublicWebURL)
	err := a.Export.Execute(r.Context(), f, out.write)
	if err == nil {
		err = out.close()
	}
	if err != nil && !out.started {
		writeError(w, err, a.Log)
		return
	}
	if err != nil {
		a.Log.Warn("exportação interrompida", "error", err)
	}
}
```

`exportWriter` guarda `started`; na primeira linha (ou em `close` sem linhas) escreve `Content-Type` (`text/csv; charset=utf-8` ou `application/json; charset=utf-8`), `Content-Disposition` (`attachment; filename="diario-sg-busca-AAAA-MM-DD.<ext>"`), `X-Total-Count`, `X-Export-Truncated` (`total > ExportLimit`), BOM + cabeçalho CSV (ou `{"total":…,"truncated":…,"items":[`); cada linha vira um registro CSV ou um objeto JSON separado por vírgula; `close` faz `Flush` (CSV) ou fecha `]}`.

Item JSON:

```go
type exportItemDTO struct {
	actHitDTO
	Body           string `json:"body"`
	ArchivedPDFURL string `json:"archived_pdf_url"`
}
```

(`Snippet` fica vazio e sai com `omitempty` só neste DTO: campo `Snippet string json:"snippet,omitempty"` sombreando o embutido.)

- [ ] **Step 4:** integração (`export_test.go`, edição `investigatorGazette`): CSV começa com BOM, cabeçalho igual a `exportHeader` unido por `;`, 3 registros, `X-Total-Count: 3`, `X-Export-Truncated: false`, coluna `pdf_arquivado` = `https://web.exemplo/api/v1/gazettes/<id>/pdf#page=1`; `min_value=40000` → 2 registros; JSON com `total=3`, `truncated=false`, `body` do contrato 3 completo; `format=xml` → 400. `make lint test test-integration` → PASS.
- [ ] **Step 5:** commit `feat(api): exportação da busca em CSV e JSON`.

### Task 3: Download no front

**Files:**
- Modify: `apps/web/src/searchState.ts` (`exportUrl`), `searchState.test.ts`
- Modify: `apps/web/src/SearchPage.tsx`, `styles.css`

- [ ] **Step 1: Teste que falha**

```ts
describe("exportUrl", () => {
  it("leva os filtros sem paginação", () => {
    const url = exportUrl({ ...EMPTY_STATE, q: "merenda", organ: "SEMED", min: "1.000", page: 3 }, "csv")!;
    const p = new URLSearchParams(url.split("?")[1]);
    expect(url.startsWith("/api/v1/acts/export?")).toBe(true);
    expect(p.get("format")).toBe("csv");
    expect(p.get("organ")).toBe("SEMED");
    expect(p.get("min_value")).toBe("1000");
    expect(p.has("limit") || p.has("offset")).toBe(false);
  });
});
```

- [ ] **Step 2:** `npm test` → FAIL.
- [ ] **Step 3:** `exportUrl(s, format)` usa `apiParams(s, 1)`, apaga `limit` e `offset`, põe `format`; devolve `null` se `apiParams` for `null`. Em `SearchPage`, com `result.total > 0`, `<p className="export">Baixar resultado: <a href=…csv download>CSV</a> · <a …json download>JSON</a>{total > 10000 && " (só os 10.000 primeiros)"}</p>`.
- [ ] **Step 4:** `npm test && npm run typecheck`; baixar um CSV na base local e abrir com `python3 -c "import csv…"` (separador `;`, BOM).
- [ ] **Step 5:** commit `feat(web): baixar o resultado da busca em CSV ou JSON`.

### Task 4: Documentação

- [ ] README (rota de exportação, colunas), decisões (CSV pt-BR, fórmula, teto, streaming), roadmap. Commit `docs: exportação da busca`.
