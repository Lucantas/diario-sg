# Ato citável (1b) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expor página e SHA-256 de cada ato, servir o PDF arquivado pela API e oferecer no front o link na página certa e uma citação pronta.

**Architecture:** `ActHit` ganha o checksum da edição; as consultas leem `page_start`/`page_end`. Um caso de uso novo, `GetGazettePDF`, junta `GazetteRepository.FindByID` e `FileStorage.Get` e a rota `GET /v1/gazettes/{id}/pdf` faz streaming com `ETag`. O front ganha `src/citation.ts` (funções puras, Vitest) e o bloco "Citar".

**Tech Stack:** Go 1.23, PostgreSQL 16, cliente GCS REST de `pkg/gcp`, React 18, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-22-ato-citavel-design.md`

## Global Constraints

- Zero comentários novos no código; porquê no commit ou em `docs/`.
- Sem migration.
- Página nula nunca vira página 1: JSON `null`, link sem `#page`, citação sem "p.".
- Conventional commits em português, sem link de sessão de agente.
- Antes de concluir: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`, `terraform fmt -check -recursive infra` e `terraform validate` (dev e prod).

---

### Task 1: Página e hash na busca

**Files:**
- Modify: `services/api/internal/core/domain/act.go` (`ActHit.Checksum`)
- Modify: `services/api/internal/adapters/postgres/acts.go` (`Search`, `ListByGazette`), `entities.go` (`ReportByEntity`)
- Modify: `services/api/internal/presentation/http/dto.go`, `router.go` (`getGazette`)
- Create: `services/api/internal/integration/citation_test.go`

**Interfaces:**
- Produces: JSON `page_start`/`page_end` (`*int`, `null` quando 0) e `pdf_sha256` em `actHitDTO`; `pdf_sha256` em `gazetteDTO`; `page_start`, `page_end`, `organ`, `organ_name` em `actDTO`; helper `pageOrNil(int) *int`.

- [ ] **Step 1: Teste de integração que falha** — indexa `organGazette` (de `organ_test.go`), zera a página de um ato com `UPDATE acts SET page_start = NULL, page_end = NULL WHERE position = 0`, e confere em `/v1/acts`: atos com `page_start == 1`, o de posição 0 com `null`, e `pdf_sha256` igual ao checksum; em `/v1/gazettes/{id}` o `pdf_sha256` e a página de cada ato.
- [ ] **Step 2:** `make test-integration` → FAIL.
- [ ] **Step 3:** SQL lê `a.page_start`, `a.page_end` com `coalesce(…, 0)` e `g.checksum`; DTO:

```go
func pageOrNil(p int) *int {
	if p <= 0 {
		return nil
	}
	return &p
}
```

- [ ] **Step 4:** `make test-integration` → PASS.
- [ ] **Step 5:** commit `feat(api): página e SHA-256 do PDF em cada ato`.

### Task 2: PDF arquivado

**Files:**
- Create: `services/api/internal/core/usecase/gazette_pdf.go`, `gazette_pdf_test.go`
- Create: `services/api/internal/presentation/http/pdf.go`, `pdf_test.go`
- Modify: `services/api/internal/presentation/http/router.go` (`API.PDF`, rota)
- Modify: `services/api/internal/config/config.go` (API exige `GAZETTE_BUCKET`)
- Modify: `services/api/cmd/api/main.go` (storage + caso de uso)
- Modify: `services/api/internal/integration/citation_test.go`

**Interfaces:**
- Produces: `usecase.NewGetGazettePDF(ports.GazetteRepository, ports.FileStorage) *GetGazettePDF`; `Gazette(ctx, id string) (domain.Gazette, error)`; `Open(ctx, g domain.Gazette) (io.ReadCloser, error)`; `pdfFilename(domain.Gazette) string`. Duas chamadas para o handler responder `304` sem ler o bucket (leitura em ARCHIVE é cobrada).

- [ ] **Step 1: Testes que falham**

```go
func TestGetGazettePDFStreamsTheArchivedFile(t *testing.T) {
	gaz := newMemGazettes()
	g := &domain.Gazette{Checksum: "abc", StoragePath: "2026/09/18.pdf", PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)}
	_ = gaz.SaveWithActs(context.Background(), g, nil)

	uc := NewGetGazettePDF(gaz, memStorage{})

	got, err := uc.Gazette(context.Background(), g.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, err := uc.Open(context.Background(), got)
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	b, _ := io.ReadAll(body)
	if got.Checksum != "abc" || string(b) != "PDF" {
		t.Fatalf("veio %+v %q", got, b)
	}
}

func TestGetGazettePDFUnknownGazette(t *testing.T) {
	_, err := NewGetGazettePDF(newMemGazettes(), memStorage{}).Gazette(context.Background(), "nada")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, veio %v", err)
	}
}
```

```go
func TestPDFFilename(t *testing.T) {
	day := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		g    domain.Gazette
		want string
	}{
		{domain.Gazette{PublishedAt: day, EditionNumber: "1771"}, "diario-sg-2026-09-18-1771.pdf"},
		{domain.Gazette{PublishedAt: day, EditionNumber: "1772", IsExtra: true}, "diario-sg-2026-09-18-1772-extra.pdf"},
		{domain.Gazette{PublishedAt: day}, "diario-sg-2026-09-18-s-n.pdf"},
	}
	for _, c := range cases {
		if got := pdfFilename(c.g); got != c.want {
			t.Errorf("esperava %s, veio %s", c.want, got)
		}
	}
}
```

- [ ] **Step 2:** `go test ./internal/core/usecase/ ./internal/presentation/http/` → FAIL.
- [ ] **Step 3: Implementação**

```go
type GetGazettePDF struct {
	gazettes ports.GazetteRepository
	storage  ports.FileStorage
}

func NewGetGazettePDF(g ports.GazetteRepository, s ports.FileStorage) *GetGazettePDF {
	return &GetGazettePDF{gazettes: g, storage: s}
}

func (uc *GetGazettePDF) Gazette(ctx context.Context, id string) (domain.Gazette, error) {
	return uc.gazettes.FindByID(ctx, id)
}

func (uc *GetGazettePDF) Open(ctx context.Context, g domain.Gazette) (io.ReadCloser, error) {
	body, err := uc.storage.Get(ctx, g.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("pdf da edição %s: %w", g.ID, err)
	}
	return body, nil
}
```

Handler (`pdf.go`):

```go
func (a *API) gazettePDF(w http.ResponseWriter, r *http.Request) {
	g, err := a.PDF.Gazette(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	etag := `"` + g.Checksum + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	body, err := a.PDF.Open(r.Context(), g)
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+pdfFilename(g)+`"`)
	if _, err := io.Copy(w, body); err != nil {
		a.Log.Warn("envio do pdf interrompido", "gazette", g.ID, "error", err)
	}
}

func pdfFilename(g domain.Gazette) string {
	number := g.EditionNumber
	if number == "" {
		number = "s-n"
	}
	name := "diario-sg-" + g.PublishedAt.Format(time.DateOnly) + "-" + number
	if g.IsExtra {
		name += "-extra"
	}
	return name + ".pdf"
}
```

Config: `RoleAPI` exige `GAZETTE_BUCKET`. `cmd/api` cria `gcp.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))`.

- [ ] **Step 4:** integração: `GET /v1/gazettes/{id}/pdf` → 200, corpo igual ao do storage, `ETag` com o checksum, `Content-Disposition` com `diario-sg-2026-09-18-7.pdf`; com `If-None-Match` → 304 sem corpo; id inexistente (uuid válido) → 404. `make lint test test-integration` → PASS.
- [ ] **Step 5:** commit `feat(api): PDF arquivado servido pela API`.

### Task 3: Infra da API

**Files:**
- Modify: `infra/stack/main.tf` (`google_storage_bucket_iam_member.api_reads`, `GAZETTE_BUCKET` no módulo `api`)

- [ ] **Step 1:**

```hcl
resource "google_storage_bucket_iam_member" "api_reads" {
  bucket = google_storage_bucket.gazettes.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.sa["api"].email}"
}
```

e no módulo `api`: `env = merge(local.email_env, { GAZETTE_BUCKET = google_storage_bucket.gazettes.name })`.
- [ ] **Step 2:** `terraform fmt -check -recursive infra` e `validate` (dev e prod).
- [ ] **Step 3:** commit `feat(infra): API lê o bucket de edições`.

### Task 4: Citação e links no front

**Files:**
- Modify: `apps/web/package.json` (`vitest`, script `test`)
- Create: `apps/web/src/citation.ts`, `apps/web/src/citation.test.ts`
- Modify: `apps/web/src/api.ts` (`page_start`, `page_end`, `pdf_sha256`)
- Modify: `apps/web/src/components.tsx` (links com página, "cópia arquivada", "Citar")
- Modify: `apps/web/src/styles.css`
- Modify: `.github/workflows/ci.yml` (`npm test` no job web)

**Interfaces:**
- Produces: `pageFragment(hit): string` (`#page=N` ou `""`), `pageLabel(hit): string` (`p. 3`, `p. 3-4` ou `""`), `archivedPdfUrl(origin, hit): string`, `formatCitation(hit, origin, accessed: Date): string`.

- [ ] **Step 1: Testes que falham** (`citation.test.ts`)

```ts
import { describe, expect, it } from "vitest";
import { formatCitation, pageLabel } from "./citation";

const hit = {
  id: "a1", gazette_id: "g1", type: "portaria", title: "PORTARIA Nº 10/2026", organ: "SEMAD",
  organ_name: "", snippet: "", edition_number: "1771", published_at: "2026-09-18", is_extra: false,
  source_url: "https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf", cnpjs: [],
  page_start: 3, page_end: 4, pdf_sha256: "ab".repeat(32),
} as const;

describe("pageLabel", () => {
  it("mostra uma página ou o intervalo", () => {
    expect(pageLabel({ ...hit, page_end: 3 })).toBe("p. 3");
    expect(pageLabel(hit)).toBe("p. 3-4");
    expect(pageLabel({ ...hit, page_start: null, page_end: null })).toBe("");
  });
});

describe("formatCitation", () => {
  const accessed = new Date(2026, 8, 22);

  it("cita edição, data, página, links e hash", () => {
    expect(formatCitation(hit, "https://diario.exemplo", accessed)).toBe(
      "SÃO GONÇALO (RJ). Diário Oficial do Município de São Gonçalo, ed. 1771, 18 set. 2026, p. 3-4. " +
      "PORTARIA Nº 10/2026. Disponível em: <https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf#page=3>. " +
      "Cópia arquivada em: <https://diario.exemplo/api/v1/gazettes/g1/pdf#page=3>. " +
      `SHA-256 do PDF: ${"ab".repeat(32)}. Acesso em: 22 set. 2026.`,
    );
  });

  it("marca edição extra, sem número e sem página", () => {
    const text = formatCitation(
      { ...hit, edition_number: "", is_extra: true, page_start: null, page_end: null },
      "https://diario.exemplo", accessed,
    );
    expect(text).toContain("ed. s/n (extra), 18 set. 2026. PORTARIA");
    expect(text).toContain("<https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf>");
    expect(text).not.toContain("#page");
  });
});
```

- [ ] **Step 2:** `npm install -D vitest` e `npm test` → FAIL (módulo inexistente).
- [ ] **Step 3: Implementação** (`citation.ts`)

```ts
import { ActHit } from "./api";

const MONTHS = ["jan.", "fev.", "mar.", "abr.", "maio", "jun.", "jul.", "ago.", "set.", "out.", "nov.", "dez."];

type Citable = Pick<ActHit, "gazette_id" | "title" | "edition_number" | "published_at" | "is_extra"
  | "source_url" | "page_start" | "page_end" | "pdf_sha256">;

export function pageFragment(hit: Pick<Citable, "page_start">) {
  return hit.page_start ? `#page=${hit.page_start}` : "";
}

export function pageLabel(hit: Pick<Citable, "page_start" | "page_end">) {
  if (!hit.page_start) return "";
  if (!hit.page_end || hit.page_end === hit.page_start) return `p. ${hit.page_start}`;
  return `p. ${hit.page_start}-${hit.page_end}`;
}

export function archivedPdfUrl(origin: string, hit: Pick<Citable, "gazette_id" | "page_start">) {
  return `${origin}/api/v1/gazettes/${hit.gazette_id}/pdf${pageFragment(hit)}`;
}

function abntDate(d: Date) {
  return `${d.getDate()} ${MONTHS[d.getMonth()]} ${d.getFullYear()}`;
}

export function formatCitation(hit: Citable, origin: string, accessed: Date) {
  const published = new Date(`${hit.published_at}T12:00:00`);
  const edition = `ed. ${hit.edition_number || "s/n"}${hit.is_extra ? " (extra)" : ""}`;
  const where = [edition, abntDate(published), pageLabel(hit)].filter(Boolean).join(", ");
  return [
    "SÃO GONÇALO (RJ). Diário Oficial do Município de São Gonçalo, " + where + ".",
    `${hit.title}.`,
    `Disponível em: <${hit.source_url}${pageFragment(hit)}>.`,
    `Cópia arquivada em: <${archivedPdfUrl(origin, hit)}>.`,
    `SHA-256 do PDF: ${hit.pdf_sha256}.`,
    `Acesso em: ${abntDate(accessed)}.`,
  ].join(" ");
}
```

`api.ts`: `page_start: number | null; page_end: number | null; pdf_sha256: string;`.
`components.tsx`: link da edição com `pageFragment`; `pageLabel` + link "cópia arquivada"; botão "Citar" (estado local) que mostra `<div className="cite"><p>{texto}</p><button>Copiar</button></div>`; copiar usa `navigator.clipboard.writeText` e, se falhar, seleciona o parágrafo.
CI: `- run: npm test` depois do typecheck.

- [ ] **Step 4:** `npm test && npm run typecheck`; conferir na base local que o link abre o PDF na página certa.
- [ ] **Step 5:** commit `feat(web): link na página do ato, cópia arquivada e citação`.

### Task 5: Documentação

- [ ] README (API: `/v1/gazettes/{id}/pdf`, campos novos), roadmap (1b feita), decisões (API serve o PDF, cache imutável, id de ato não é permanente). Commit `docs: ato citável`.
