# Ato localizável (Entrega 1a) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Gravar a página (início e fim) e a sigla do órgão de cada ato, e ter um comando que reprocessa edições já indexadas com o parser atual.

**Architecture:** O parser passa a carregar a página em cada linha (`[]line` no lugar de `[]string`) e o órgão corrente ao percorrer as seções. A migration 005 adiciona `page_start`, `page_end` e `organ` em `acts`. Um caso de uso novo, `ReindexGazettes`, relê o PDF do bucket e troca os atos de cada edição numa transação (`ReplaceActs`), exposto por `cmd/reindex` e `make reindex`.

**Tech Stack:** Go 1.22+, PostgreSQL 16 (`lib/pq`), `pdftotext` (poppler), emulador do GCS (`fake-gcs-server`) no ambiente local.

**Spec:** `docs/superpowers/specs/2026-09-22-ato-localizavel-design.md`

## Global Constraints

- Zero comentários novos no código, inclusive em SQL (regra do dono do repositório). Comentários que já existem ficam; quando o código deles muda, são ajustados para continuar verdadeiros. O "porquê" vai na mensagem de commit.
- A segmentação dos atos não muda: mesmos atos, mesmos títulos, mesmos tipos. Os testes existentes do parser não podem ser alterados para passar.
- Reindexação não publica `gazette.indexed` e não envia alertas.
- Mensagens de log e de erro em português, como o resto do código.
- Comandos: `make lint`, `make test`, `make test-integration` (precisa de `make up`), a partir da raiz do repositório.

---

### Task 1: Página de cada ato no parser

**Files:**
- Modify: `services/api/internal/core/domain/act.go` (struct `Act`)
- Modify: `services/api/internal/adapters/parser/clean.go` (`splitLines`, `stripPageNoise`, `lastNonEmpty`, `dropPreamble`, `joinSplitHeaders`)
- Modify: `services/api/internal/adapters/parser/headers.go` (`isOrganSection`)
- Modify: `services/api/internal/adapters/parser/regex.go` (`Parse`, `segment`)
- Create: `services/api/internal/adapters/parser/pages_test.go`

**Interfaces:**
- Produces: `domain.Act.PageStart int`, `domain.Act.PageEnd int` (sempre ≥ 1 quando o ato vem do parser); tipo interno `line{text string; page int}`; `segment.add(ls ...line)`; `joinTexts(ls []line, sep string) string`.

- [ ] **Step 1: Escrever os testes que falham**

Criar `services/api/internal/adapters/parser/pages_test.go`:

```go
package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func pageSpans(acts []domain.Act) []string {
	out := make([]string, 0, len(acts))
	for _, a := range acts {
		out = append(out, fmt.Sprintf("%s %d-%d", a.Title, a.PageStart, a.PageEnd))
	}
	return out
}

func TestParse_PagesFollowFormFeeds(t *testing.T) {
	text := "DECRETO Nº 1/2026\nDispõe sobre o horário.\n\fcontinua o decreto 1.\n" +
		"DECRETO Nº 2/2026\nDispõe sobre a limpeza.\n\fDECRETO Nº 3/2026\nDispõe sobre a feira."

	got := pageSpans(New().Parse(text))

	want := []string{"DECRETO Nº 1/2026 1-2", "DECRETO Nº 2/2026 2-2", "DECRETO Nº 3/2026 3-3"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("esperava %v, veio %v", want, got)
	}
}

func TestParse_TextWithoutFormFeedIsPageOne(t *testing.T) {
	got := pageSpans(New().Parse("DECRETO Nº 1/2026\nDispõe sobre o horário."))

	if len(got) != 1 || got[0] != "DECRETO Nº 1/2026 1-1" {
		t.Errorf("veio %v", got)
	}
}

func TestParse_SplitHeaderAcrossPagesKeepsTheFirstPage(t *testing.T) {
	text := "TERMO\nDE\n\fAPREENSÃO\nADMINISTRATIVA\nNº\n5/2026\nApreendido o veículo de placa ABC1D23."

	got := pageSpans(New().Parse(text))

	want := "TERMO DE APREENSÃO ADMINISTRATIVA Nº 5/2026 1-2"
	if len(got) != 1 || got[0] != want {
		t.Errorf("esperava [%s], veio %v", want, got)
	}
}

func TestRealEditions_ActPagesMatchTheText(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "..", "testdata", "editions", "*.txt"))
	if len(files) == 0 {
		t.Skip("sem edições reais em testdata/editions (rode ./scripts/fetch-editions.sh)")
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		pages := strings.Split(string(raw), "\f")
		onPage := make([]map[string]bool, len(pages))
		anywhere := map[string]bool{}
		for i, p := range pages {
			onPage[i] = map[string]bool{}
			for _, l := range strings.Split(p, "\n") {
				l = strings.TrimSpace(l)
				onPage[i][l] = true
				anywhere[l] = true
			}
		}
		name := filepath.Base(f)
		for _, a := range New().Parse(string(raw)) {
			if a.PageStart < 1 || a.PageEnd < a.PageStart || a.PageEnd > len(pages) {
				t.Errorf("%s: %q com páginas inválidas %d-%d (edição tem %d)", name, a.Title, a.PageStart, a.PageEnd, len(pages))
				continue
			}
			bodyLines := strings.Split(a.Body, "\n")
			first := strings.TrimSpace(bodyLines[0])
			last := strings.TrimSpace(bodyLines[len(bodyLines)-1])
			if anywhere[first] && !onPage[a.PageStart-1][first] {
				t.Errorf("%s: %q começa em %q, que não está na página %d", name, a.Title, first, a.PageStart)
			}
			if anywhere[last] && !onPage[a.PageEnd-1][last] {
				t.Errorf("%s: %q termina em %q, que não está na página %d", name, a.Title, last, a.PageEnd)
			}
		}
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `cd services/api && go test ./internal/adapters/parser/ -run 'Pages|PageOne|SplitHeaderAcross|ActPagesMatch' -v`
Expected: FAIL de compilação (`a.PageStart undefined`).

- [ ] **Step 3: Adicionar os campos ao domínio**

Em `services/api/internal/core/domain/act.go`, na struct `Act`, depois de `Position  int`:

```go
	PageStart int
	PageEnd   int
```

(`gofmt` realinha a struct.)

- [ ] **Step 4: Levar a página pelas linhas em `clean.go`**

Substituir `splitLines`, `stripPageNoise`, `lastNonEmpty`, `dropPreamble` e `joinSplitHeaders` por:

```go
type line struct {
	text string
	page int
}

func joinTexts(ls []line, sep string) string {
	parts := make([]string, len(ls))
	for i, l := range ls {
		parts[i] = l.text
	}
	return strings.Join(parts, sep)
}

// splitLines normaliza quebras e remove espaços nas pontas de cada linha.
func splitLines(text string) []line {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out []line
	for p, page := range strings.Split(text, "\f") {
		for _, l := range strings.Split(page, "\n") {
			out = append(out, line{text: strings.TrimSpace(l), page: p + 1})
		}
	}
	return out
}

// stripPageNoise remove cabeçalho e rodapé de página. O número da página é
// uma linha só com dígitos imediatamente antes da URL do site; só é
// removido nesse contexto, porque tabelas também têm linhas só com dígitos.
func stripPageNoise(lines []line) []line {
	out := make([]line, 0, len(lines))
	for _, l := range lines {
		if pageNoiseRe.MatchString(l.text) {
			continue
		}
		if siteURLRe.MatchString(l.text) {
			if n := lastNonEmpty(out); n >= 0 && pageNumberRe.MatchString(out[n].text) {
				out = out[:n]
			}
			continue
		}
		out = append(out, l)
	}
	return out
}

func lastNonEmpty(lines []line) int {
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].text != "" {
			return i
		}
	}
	return -1
}
```

`dropPreamble` fica igual trocando o tipo e os acessos:

```go
func dropPreamble(lines []line) []line {
	for i, l := range lines {
		if preambleMarkers[l.text] {
			return lines[i+1:]
		}
	}
	for i, l := range lines {
		if startsAct(l.text) {
			return lines[i:]
		}
	}
	return lines
}
```

`joinSplitHeaders` mantém a lógica; a linha reunida fica com a página da primeira:

```go
func joinSplitHeaders(lines []line) []line {
	out := make([]line, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if !splitHeaderStart[l.text] {
			out = append(out, l)
			continue
		}
		j := i + 1
		for j < len(lines) && j-i < maxJoinedTokens && singleTokenRe.MatchString(lines[j].text) && !splitHeaderStart[lines[j].text] {
			j++
		}
		if j-i < 3 {
			out = append(out, l)
			continue
		}
		out = append(out, line{text: joinTexts(lines[i:j], " "), page: l.page})
		i = j - 1
	}
	return out
}
```

Manter os comentários que já existem sobre `dropPreamble` e `joinSplitHeaders`.

- [ ] **Step 5: `isOrganSection` em `headers.go`**

```go
func isOrganSection(lines []line, i int) bool {
	if !organRe.MatchString(lines[i].text) || isHeader(lines[i].text) {
		return false
	}
	for j := i + 1; j < len(lines) && j <= i+2; j++ {
		if lines[j].text == "" {
			continue
		}
		return startsAct(lines[j].text)
	}
	return false
}
```

- [ ] **Step 6: `Parse` e `segment` em `regex.go`**

Substituir o corpo de `Parse` e a struct `segment` com seus métodos `close` e `act`:

```go
func (Regex) Parse(text string) []domain.Act {
	lines := dropPreamble(joinSplitHeaders(stripPageNoise(splitLines(text))))

	var acts []domain.Act
	var current *segment
	flush := func() {
		if current == nil {
			return
		}
		if act, ok := current.act(len(acts)); ok {
			acts = append(acts, act)
		}
		current = nil
	}

	for i := 0; i < len(lines); i++ {
		l := lines[i]
		switch {
		case isOrganSection(lines, i), isContinuation(l.text):
			flush()
		case isPortariaTrailer(l.text):
			if current == nil {
				current = &segment{}
			}
			current.close(l)
			flush()
		case isPortariaVerb(l.text):
			flush()
			current = &segment{title: l.text}
			current.add(l)
		case isHeader(l.text):
			flush()
			title := []line{l}
			for headerContinues(l.text) && i+1 < len(lines) && lines[i+1].text != "" {
				i++
				l = lines[i]
				title = append(title, l)
			}
			current = &segment{title: joinTexts(title, " ")}
			current.add(title...)
		default:
			if current == nil {
				if l.text == "" {
					continue
				}
				current = &segment{title: l.text, orphan: true}
			}
			current.add(l)
		}
	}
	flush()
	return acts
}

type segment struct {
	title     string
	lines     []string
	orphan    bool
	pageStart int
	pageEnd   int
}

func (s *segment) add(ls ...line) {
	for _, l := range ls {
		s.lines = append(s.lines, l.text)
		if l.text == "" {
			continue
		}
		if s.pageStart == 0 {
			s.pageStart = l.page
		}
		s.pageEnd = l.page
	}
}

// close dá ao segmento o número de portaria que o encerra.
func (s *segment) close(trailer line) {
	s.title = trailer.text
	s.orphan = false
	s.add(trailer)
}
```

E no fim de `act`, o retorno passa a ser:

```go
	return domain.Act{Type: classify(title, body), Title: title, Body: body, Position: position,
		PageStart: s.pageStart, PageEnd: s.pageEnd}, true
```

- [ ] **Step 7: Rodar os testes do parser**

Run: `cd services/api && go test ./internal/adapters/parser/ -v 2>&1 | grep -E '^(=== RUN|--- FAIL|--- PASS|ok|FAIL)'`
Expected: todos PASS, inclusive `TestParseRealFixtures`, `TestMeasureRealEditions` (51 atos em 2026_09_18) e `TestRealEditions_ActPagesMatchTheText` (se `testdata/editions` existir localmente; senão SKIP).

Se `TestRealEditions_ActPagesMatchTheText` falhar, ler a mensagem: ela diz qual linha não está na página atribuída. Corrigir o parser, não o teste.

- [ ] **Step 8: Lint e testes do módulo**

Run: `make lint && make test`
Expected: sem saída de erro; todos os pacotes `ok`.

- [ ] **Step 9: Commit**

```bash
git add services/api/internal/core/domain/act.go services/api/internal/adapters/parser/
git commit -m "feat(parser): página de início e fim de cada ato"
```

---

### Task 2: Órgão de cada ato no parser

**Files:**
- Modify: `services/api/internal/core/domain/act.go` (struct `Act`)
- Modify: `services/api/internal/adapters/parser/headers.go` (lista `nonOrganSections`)
- Modify: `services/api/internal/adapters/parser/regex.go` (`Parse`, `segment`, `act`)
- Create: `services/api/internal/adapters/parser/organ_test.go`

**Interfaces:**
- Consumes: `line`, `segment.add` (Task 1).
- Produces: `domain.Act.Organ string` (sigla ou `""`).

- [ ] **Step 1: Escrever o teste que falha**

Criar `services/api/internal/adapters/parser/organ_test.go`:

```go
package parser

import (
	"strings"
	"testing"
)

func TestParse_OrganComesFromTheSectionAcronym(t *testing.T) {
	text := "ATOS DO PREFEITO\n" +
		"DECRETO Nº 1/2026\nDispõe sobre o horário.\n" +
		"SEMAD\nPORTARIA Nº 10/2026\nNomeia servidor para a função.\n" +
		"PORTARIA Nº 11/2026\nExonera servidor da função.\n" +
		"FMS\nEXTRATO DO CONTRATO Nº 3/2026\nObjeto: medicamentos.\n" +
		"ANEXO\nDECRETO Nº 2/2026\nDispõe sobre a feira."

	acts := New().Parse(text)

	var got []string
	for _, a := range acts {
		got = append(got, a.Title+"="+a.Organ)
	}
	want := []string{
		"DECRETO Nº 1/2026=",
		"PORTARIA Nº 10/2026=SEMAD",
		"PORTARIA Nº 11/2026=SEMAD",
		"EXTRATO DO CONTRATO Nº 3/2026=FMS",
		"DECRETO Nº 2/2026=FMS",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("esperava\n%v\nveio\n%v", want, got)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `cd services/api && go test ./internal/adapters/parser/ -run TestParse_OrganComesFromTheSectionAcronym -v`
Expected: FAIL de compilação (`a.Organ undefined`).

- [ ] **Step 3: Campo no domínio**

Em `domain.Act`, depois de `PageEnd   int`:

```go
	Organ     string
```

- [ ] **Step 4: Siglas que não são órgão, em `headers.go`**

Logo abaixo de `isOrganSection`:

```go
var nonOrganSections = map[string]bool{"ANEXO": true}
```

- [ ] **Step 5: Órgão corrente em `Parse` e no `segment`**

Em `regex.go`:

1. Declarar `organ := ""` logo depois de `var current *segment`.
2. Trocar o primeiro `case` por dois:

```go
		case isOrganSection(lines, i):
			flush()
			if !nonOrganSections[l.text] {
				organ = l.text
			}
		case isContinuation(l.text):
			flush()
```

3. Todo `&segment{...}` criado em `Parse` recebe `organ: organ` (são quatro: o do trailer, o do verbo, o do cabeçalho e o órfão). Exemplo: `current = &segment{title: l.text, organ: organ}`.
4. Na struct `segment`, adicionar `organ string`.
5. No retorno de `act`, adicionar `Organ: s.organ`:

```go
	return domain.Act{Type: classify(title, body), Title: title, Body: body, Position: position,
		PageStart: s.pageStart, PageEnd: s.pageEnd, Organ: s.organ}, true
```

- [ ] **Step 6: Rodar os testes do parser**

Run: `cd services/api && go test ./internal/adapters/parser/ -v 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'`
Expected: `ok`; nenhum `--- FAIL`.

- [ ] **Step 7: Lint, testes e commit**

```bash
make lint && make test
git add services/api/internal/core/domain/act.go services/api/internal/adapters/parser/
git commit -m "feat(parser): sigla do órgão de cada ato"
```

---

### Task 3: Gravar página e órgão (migration 005)

**Files:**
- Create: `services/api/migrations/005_act_location.sql`
- Modify: `services/api/internal/adapters/postgres/gazettes.go` (`SaveWithActs`; nova função `insertActs`)
- Modify: `services/api/internal/integration/e2e_test.go` (novo passo depois do passo 1)

**Interfaces:**
- Consumes: `domain.Act.PageStart`, `PageEnd`, `Organ` (Tasks 1-2).
- Produces: `func insertActs(ctx context.Context, tx *sql.Tx, gazetteID string, acts []domain.Act) error` em `adapters/postgres/gazettes.go`, usada também pela Task 5.

- [ ] **Step 1: Escrever o teste de integração que falha**

Em `services/api/internal/integration/e2e_test.go`, logo depois do laço de indexação do passo 1 (antes de `api := &httpapi.API{...}`), inserir:

```go
	var withoutPage, total int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*) FILTER (WHERE page_start IS DISTINCT FROM 1 OR page_end IS DISTINCT FROM 1 OR organ <> ''), count(*)
		FROM acts`).Scan(&withoutPage, &total); err != nil {
		t.Fatal(err)
	}
	if total != 5 || withoutPage != 0 {
		t.Fatalf("esperava 5 atos com página 1 e órgão vazio, veio total=%d fora do esperado=%d", total, withoutPage)
	}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `make test-integration`
Expected: FAIL com `column "page_start" does not exist`.

- [ ] **Step 3: Migration**

Criar `services/api/migrations/005_act_location.sql`:

```sql
ALTER TABLE acts
    ADD COLUMN page_start int,
    ADD COLUMN page_end   int,
    ADD COLUMN organ      text NOT NULL DEFAULT '';
CREATE INDEX acts_organ_idx ON acts (organ);
```

- [ ] **Step 4: Extrair `insertActs` e gravar as colunas**

Em `gazettes.go`, mover o trecho de `SaveWithActs` que prepara e executa os INSERTs de atos e entidades para:

```go
func insertActs(ctx context.Context, tx *sql.Tx, gazetteID string, acts []domain.Act) error {
	insertAct, err := tx.PrepareContext(ctx, `
		INSERT INTO acts (gazette_id, type, title, body, position, page_start, page_end, organ)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`)
	if err != nil {
		return err
	}
	defer insertAct.Close()
	insertEntity, err := tx.PrepareContext(ctx, `
		INSERT INTO act_entities (act_id, kind, value, normalized) VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING`)
	if err != nil {
		return err
	}
	defer insertEntity.Close()
	for _, a := range acts {
		var actID string
		if err := insertAct.QueryRowContext(ctx, gazetteID, string(a.Type), a.Title, a.Body, a.Position,
			a.PageStart, a.PageEnd, a.Organ).Scan(&actID); err != nil {
			return fmt.Errorf("ato %d: %w", a.Position, err)
		}
		for _, e := range a.Entities {
			if _, err := insertEntity.ExecContext(ctx, actID, string(e.Kind), e.Value, e.Normalized); err != nil {
				return fmt.Errorf("ato %d, entidade %s: %w", a.Position, e.Kind, err)
			}
		}
	}
	return nil
}
```

E em `SaveWithActs`, depois do INSERT da edição (e do tratamento de `sql.ErrNoRows`), o bloco antigo vira:

```go
	if err := insertActs(ctx, tx, g.ID, acts); err != nil {
		return err
	}
	return tx.Commit()
```

- [ ] **Step 5: Rodar os testes**

Run: `make lint && make test && make test-integration`
Expected: tudo `ok`.

- [ ] **Step 6: Commit**

```bash
git add services/api/migrations/005_act_location.sql services/api/internal/adapters/postgres/gazettes.go services/api/internal/integration/e2e_test.go
git commit -m "feat(api): grava página e órgão de cada ato (migration 005)"
```

---

### Task 4: Caso de uso de reindexação

**Files:**
- Modify: `services/api/internal/core/ports/ports.go` (`GazetteRepository`)
- Create: `services/api/internal/core/usecase/parse_acts.go`
- Modify: `services/api/internal/core/usecase/index_gazette.go` (usar `parseActs`)
- Create: `services/api/internal/core/usecase/reindex_gazettes.go`
- Create: `services/api/internal/core/usecase/reindex_gazettes_test.go`
- Modify: `services/api/internal/core/usecase/fakes_test.go` (`memGazettes`)
- Modify: `services/api/internal/adapters/postgres/gazettes.go` (stubs temporários só para compilar: substituídos na Task 5)

**Interfaces:**
- Produces:
  - `ports.GazetteRepository.ListByPeriod(ctx context.Context, from, to time.Time) ([]domain.Gazette, error)` — edições com `published_at` entre `from` e `to` (inclusive), ordenadas por `published_at`, `source_url`.
  - `ports.GazetteRepository.ReplaceActs(ctx context.Context, gazetteID, editionNumber string, acts []domain.Act) error`.
  - `usecase.ReindexResult{Found, Reindexed, Failed int}` com tags JSON `found`, `reindexed`, `failed`.
  - `usecase.NewReindexGazettes(g ports.GazetteRepository, s ports.FileStorage, e ports.TextExtractor, p ports.ActParser, x ports.EntityExtractor) *ReindexGazettes`.
  - `(*ReindexGazettes).Execute(ctx context.Context, from, to time.Time) (ReindexResult, error)`.

- [ ] **Step 1: Escrever os testes que falham**

Criar `services/api/internal/core/usecase/reindex_gazettes_test.go`:

```go
package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type pathStorage map[string]string

func (s pathStorage) Get(_ context.Context, path string) (io.ReadCloser, error) {
	text, ok := s[path]
	if !ok {
		return nil, errors.New("objeto não encontrado")
	}
	return io.NopCloser(strings.NewReader(text)), nil
}

type echoExtractor struct{}

func (echoExtractor) Extract(_ context.Context, r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	return string(b), err
}

func day(d int) time.Time { return time.Date(2026, 9, d, 0, 0, 0, 0, time.UTC) }

func seed(repo *memGazettes, id string, published time.Time, number string, acts ...domain.Act) {
	repo.saved[id] = domain.Gazette{ID: id, EditionNumber: number, PublishedAt: published, StoragePath: id + ".pdf"}
	repo.acts[id] = acts
}

func TestReindexGazettes_ReplacesActsInThePeriodWithoutPublishing(t *testing.T) {
	repo := newMemGazettes()
	seed(repo, "a", day(10), "1", domain.Act{Title: "antigo"})
	seed(repo, "b", day(11), "", domain.Act{Title: "antigo"})
	seed(repo, "fora", day(20), "9", domain.Act{Title: "antigo"})
	storage := pathStorage{"a.pdf": "PORTARIA 1\nDECRETO 2 CNPJ", "b.pdf": "EDIÇÃO 1771\nPORTARIA 3", "fora.pdf": "X"}
	uc := NewReindexGazettes(repo, storage, echoExtractor{}, lineParser{}, cnpjExtractor{})

	res, err := uc.Execute(context.Background(), day(10), day(11))

	if err != nil {
		t.Fatal(err)
	}
	if res != (ReindexResult{Found: 2, Reindexed: 2}) {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if got := repo.acts["a"]; len(got) != 2 || got[0].Title != "PORTARIA 1" || len(got[1].Entities) != 1 {
		t.Fatalf("atos de a não foram trocados com entidades: %+v", got)
	}
	if repo.saved["a"].EditionNumber != "1" || repo.saved["b"].EditionNumber != "1771" {
		t.Fatalf("número: sem número no texto mantém o antigo, com número atualiza: a=%q b=%q",
			repo.saved["a"].EditionNumber, repo.saved["b"].EditionNumber)
	}
	if repo.acts["fora"][0].Title != "antigo" {
		t.Fatal("edição fora do período não pode ser tocada")
	}
}

func TestReindexGazettes_FailureKeepsOldActsAndContinues(t *testing.T) {
	repo := newMemGazettes()
	seed(repo, "sem-pdf", day(10), "1", domain.Act{Title: "antigo"})
	seed(repo, "ok", day(11), "2", domain.Act{Title: "antigo"})
	uc := NewReindexGazettes(repo, pathStorage{"ok.pdf": "PORTARIA 1"}, echoExtractor{}, lineParser{}, cnpjExtractor{})

	res, err := uc.Execute(context.Background(), day(10), day(11))

	if err == nil || !strings.Contains(err.Error(), "sem-pdf.pdf") {
		t.Fatalf("esperava erro citando a edição que falhou, veio %v", err)
	}
	if res != (ReindexResult{Found: 2, Reindexed: 1, Failed: 1}) {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if repo.acts["sem-pdf"][0].Title != "antigo" || repo.acts["ok"][0].Title != "PORTARIA 1" {
		t.Fatalf("falha não pode apagar atos nem parar as demais: %+v", repo.acts)
	}
}

func TestReindexGazettes_RejectsInvertedPeriod(t *testing.T) {
	uc := NewReindexGazettes(newMemGazettes(), pathStorage{}, echoExtractor{}, lineParser{}, cnpjExtractor{})

	_, err := uc.Execute(context.Background(), day(11), day(10))

	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("esperava ErrInvalidInput, veio %v", err)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `cd services/api && go test ./internal/core/usecase/ -run Reindex -v`
Expected: FAIL de compilação (`undefined: NewReindexGazettes`).

- [ ] **Step 3: Portas**

Em `ports.go`, adicionar `"time"` aos imports e, em `GazetteRepository`:

```go
	ListByPeriod(ctx context.Context, from, to time.Time) ([]domain.Gazette, error)
	ReplaceActs(ctx context.Context, gazetteID, editionNumber string, acts []domain.Act) error
```

- [ ] **Step 4: Fakes**

Em `fakes_test.go`, adicionar `"sort"` e `"time"` aos imports e os métodos de `memGazettes`:

```go
func (m *memGazettes) ListByPeriod(_ context.Context, from, to time.Time) ([]domain.Gazette, error) {
	var out []domain.Gazette
	for _, g := range m.saved {
		if !g.PublishedAt.Before(from) && !g.PublishedAt.After(to) {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PublishedAt.Before(out[j].PublishedAt) })
	return out, nil
}
func (m *memGazettes) ReplaceActs(_ context.Context, id, number string, acts []domain.Act) error {
	g := m.saved[id]
	g.EditionNumber = number
	m.saved[id] = g
	m.acts[id] = acts
	return nil
}
```

- [ ] **Step 5: Stubs no adapter Postgres (compilação)**

A API e o worker precisam continuar compilando até a Task 5. Em `adapters/postgres/gazettes.go`, adicionar `"time"` aos imports e:

```go
func (r *GazetteRepo) ListByPeriod(context.Context, time.Time, time.Time) ([]domain.Gazette, error) {
	return nil, errors.New("ListByPeriod: não implementado")
}

func (r *GazetteRepo) ReplaceActs(context.Context, string, string, []domain.Act) error {
	return errors.New("ReplaceActs: não implementado")
}
```

- [ ] **Step 6: Parse compartilhado**

Criar `services/api/internal/core/usecase/parse_acts.go`:

```go
package usecase

import (
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

func parseActs(p ports.ActParser, x ports.EntityExtractor, text string) []domain.Act {
	acts := p.Parse(text)
	for i := range acts {
		acts[i].Entities = x.Extract(acts[i].Body)
	}
	return acts
}
```

Em `index_gazette.go`, trocar

```go
	acts := uc.parser.Parse(text)
	for i := range acts {
		acts[i].Entities = uc.entities.Extract(acts[i].Body)
	}
```

por

```go
	acts := parseActs(uc.parser, uc.entities, text)
```

- [ ] **Step 7: Caso de uso**

Criar `services/api/internal/core/usecase/reindex_gazettes.go`:

```go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type ReindexResult struct {
	Found     int `json:"found"`
	Reindexed int `json:"reindexed"`
	Failed    int `json:"failed"`
}

type ReindexGazettes struct {
	gazettes  ports.GazetteRepository
	storage   ports.FileStorage
	extractor ports.TextExtractor
	parser    ports.ActParser
	entities  ports.EntityExtractor
}

func NewReindexGazettes(g ports.GazetteRepository, s ports.FileStorage, e ports.TextExtractor, p ports.ActParser, x ports.EntityExtractor) *ReindexGazettes {
	return &ReindexGazettes{gazettes: g, storage: s, extractor: e, parser: p, entities: x}
}

func (uc *ReindexGazettes) Execute(ctx context.Context, from, to time.Time) (ReindexResult, error) {
	if to.Before(from) {
		return ReindexResult{}, fmt.Errorf("%w: período invertido", domain.ErrInvalidInput)
	}
	gazettes, err := uc.gazettes.ListByPeriod(ctx, from, to)
	if err != nil {
		return ReindexResult{}, fmt.Errorf("listar edições: %w", err)
	}
	res := ReindexResult{Found: len(gazettes)}
	var errs []error
	for _, g := range gazettes {
		if err := uc.reindexOne(ctx, g); err != nil {
			res.Failed++
			errs = append(errs, fmt.Errorf("edição %s (%s): %w", g.PublishedAt.Format(time.DateOnly), g.StoragePath, err))
			continue
		}
		res.Reindexed++
	}
	return res, errors.Join(errs...)
}

func (uc *ReindexGazettes) reindexOne(ctx context.Context, g domain.Gazette) error {
	rc, err := uc.storage.Get(ctx, g.StoragePath)
	if err != nil {
		return fmt.Errorf("baixar: %w", err)
	}
	defer rc.Close()
	text, err := uc.extractor.Extract(ctx, rc)
	if err != nil {
		return fmt.Errorf("extrair texto: %w", err)
	}
	number := uc.parser.EditionNumber(text)
	if number == "" {
		number = g.EditionNumber
	}
	return uc.gazettes.ReplaceActs(ctx, g.ID, number, parseActs(uc.parser, uc.entities, text))
}
```

- [ ] **Step 8: Rodar os testes**

Run: `make lint && make test`
Expected: tudo `ok`, inclusive os três testes `TestReindexGazettes_*`.

- [ ] **Step 9: Commit**

```bash
git add services/api/internal/core services/api/internal/adapters/postgres/gazettes.go
git commit -m "feat(api): caso de uso de reindexação de edições"
```

---

### Task 5: Reindexação no Postgres, comando e integração

**Files:**
- Modify: `services/api/internal/adapters/postgres/gazettes.go` (implementar `ListByPeriod` e `ReplaceActs`)
- Modify: `services/api/internal/config/config.go` (`RoleReindex`)
- Create: `services/api/cmd/reindex/main.go`
- Modify: `Makefile` (alvo `reindex`, `.PHONY`)
- Create: `services/api/internal/integration/reindex_test.go`
- Modify: `README.md` (linha do `make reindex` em "Rodando localmente")

**Interfaces:**
- Consumes: `insertActs` (Task 3), portas e `NewReindexGazettes` (Task 4).
- Produces: `config.RoleReindex Role = "reindex"`; binário `cmd/reindex` com flags `-from` e `-to`; `make reindex FROM=AAAA-MM-DD TO=AAAA-MM-DD`.

- [ ] **Step 1: Escrever o teste de integração que falha**

Criar `services/api/internal/integration/reindex_test.go`:

```go
//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
	"github.com/seu-usuario/diario-sg/services/api/migrations"
)

func TestReindexReplacesActsWithoutPublishing(t *testing.T) {
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
	if _, err := db.ExecContext(ctx, `TRUNCATE gazettes, subscriptions CASCADE`); err != nil {
		t.Fatal(err)
	}
	gaz := postgres.NewGazetteRepo(db)
	pub := &recPub{}
	published := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	in := usecase.IndexGazetteInput{EditionNumber: "1", PublishedAt: published,
		SourceURL: "https://exemplo/1.pdf", StoragePath: "1.pdf", Checksum: strings.Repeat("d", 64)}
	if err := usecase.NewIndexGazette(gaz, textStore{}, passthroughExtractor{}, parser.New(), entities.New(), pub).Execute(ctx, in); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE acts SET page_start = NULL, page_end = NULL, organ = 'VELHO'`); err != nil {
		t.Fatal(err)
	}
	var entitiesBefore int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM act_entities`).Scan(&entitiesBefore); err != nil {
		t.Fatal(err)
	}

	res, err := usecase.NewReindexGazettes(gaz, textStore{}, passthroughExtractor{}, parser.New(), entities.New()).
		Execute(ctx, published, published)

	if err != nil || res != (usecase.ReindexResult{Found: 1, Reindexed: 1}) {
		t.Fatalf("reindexação: %+v %v", res, err)
	}
	var acts, stale, entitiesAfter int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*), count(*) FILTER (WHERE page_start IS NULL OR organ = 'VELHO'),
		       (SELECT count(*) FROM act_entities)
		FROM acts`).Scan(&acts, &stale, &entitiesAfter); err != nil {
		t.Fatal(err)
	}
	if acts != 5 || stale != 0 || entitiesAfter != entitiesBefore || entitiesAfter == 0 {
		t.Fatalf("esperava 5 atos refeitos com entidades (%d), veio atos=%d velhos=%d entidades=%d",
			entitiesBefore, acts, stale, entitiesAfter)
	}
	if len(pub.ids) != 1 {
		t.Fatalf("reindexação não pode publicar gazette.indexed; eventos: %v", pub.ids)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `make test-integration`
Expected: FAIL com `ListByPeriod: não implementado`.

- [ ] **Step 3: Implementar no Postgres**

Em `gazettes.go`, substituir os stubs da Task 4 por:

```go
func (r *GazetteRepo) ListByPeriod(ctx context.Context, from, to time.Time) ([]domain.Gazette, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, edition_number, published_at, is_extra, source_url, storage_path, checksum, indexed_at
		FROM gazettes
		WHERE published_at BETWEEN $1::date AND $2::date
		ORDER BY published_at, source_url`, from.Format(time.DateOnly), to.Format(time.DateOnly))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Gazette
	for rows.Next() {
		var g domain.Gazette
		if err := rows.Scan(&g.ID, &g.EditionNumber, &g.PublishedAt, &g.IsExtra, &g.SourceURL, &g.StoragePath, &g.Checksum, &g.IndexedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *GazetteRepo) ReplaceActs(ctx context.Context, gazetteID, editionNumber string, acts []domain.Act) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // sem efeito após Commit

	if _, err := tx.ExecContext(ctx, `DELETE FROM acts WHERE gazette_id = $1`, gazetteID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE gazettes SET edition_number = $2 WHERE id = $1`, gazetteID, editionNumber); err != nil {
		return err
	}
	if err := insertActs(ctx, tx, gazetteID, acts); err != nil {
		return err
	}
	return tx.Commit()
}
```

Remover o import de `errors` se ele deixar de ser usado (continua usado em `SaveWithActs` por `errors.Is`).

- [ ] **Step 4: Rodar a integração**

Run: `make test-integration`
Expected: `ok` (os dois testes).

- [ ] **Step 5: Papel de configuração**

Em `config.go`, adicionar `RoleReindex Role = "reindex"` às constantes e exigir o bucket:

```go
	if role == RoleReindex {
		required["GAZETTE_BUCKET"] = c.Bucket
	}
```

E trocar a condição do Resend para não exigir e-mail na reindexação:

```go
	if role != RoleMigrate && role != RoleReindex && c.Notifier == "resend" {
```

- [ ] **Step 6: Comando**

Criar `services/api/cmd/reindex/main.go`:

```go
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/pkg/obs"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/pdf"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

func main() {
	log := obs.NewLogger("reindex")
	fromFlag := flag.String("from", "", "primeira data (AAAA-MM-DD)")
	toFlag := flag.String("to", "", "última data (AAAA-MM-DD)")
	flag.Parse()
	from, errFrom := time.Parse(time.DateOnly, *fromFlag)
	to, errTo := time.Parse(time.DateOnly, *toFlag)
	if errFrom != nil || errTo != nil {
		log.Error("uso: reindex -from AAAA-MM-DD -to AAAA-MM-DD", "from", *fromFlag, "to", *toFlag)
		os.Exit(2)
	}
	cfg, err := config.Load(config.RoleReindex)
	if err != nil {
		log.Error("configuração inválida", "error", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("conexão falhou", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	storage := gcp.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))
	uc := usecase.NewReindexGazettes(postgres.NewGazetteRepo(db), storage, pdf.New(), parser.New(), entities.New())
	start := time.Now()
	res, err := uc.Execute(ctx, from, to)
	log.Info("reindexação finalizada", "result", res, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("edições com falha", "error", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 7: Makefile e README**

No `Makefile`, acrescentar `reindex` à linha `.PHONY` e, depois do alvo `migrate`:

```make
reindex: ## Reprocessa edições já indexadas com o parser atual: make reindex FROM=AAAA-MM-DD TO=AAAA-MM-DD
	@test -n "$(FROM)" -a -n "$(TO)" || (echo "uso: make reindex FROM=AAAA-MM-DD TO=AAAA-MM-DD"; exit 2)
	cd services/api && go run ./cmd/reindex -from "$(FROM)" -to "$(TO)"
```

No `README.md`, no bloco de "Rodando localmente", depois da linha `make run-scraper LOOKBACK_DAYS=7 ...`:

```bash
make reindex FROM=2020-01-01 TO=2026-12-31   # reprocessa edições já indexadas com o parser atual (não dispara alertas)
```

- [ ] **Step 8: Rodar tudo**

Run: `make lint && make test && make test-integration && cd services/api && go build ./cmd/...`
Expected: tudo `ok`, build sem erro.

- [ ] **Step 9: Commit**

```bash
git add services/api Makefile README.md
git commit -m "feat(api): comando make reindex para reprocessar edições"
```

---

### Task 6: Validação na base local

**Files:**
- Modify: `docs/roadmap.md` (Entrega 1: marcar a 1a)

**Interfaces:**
- Consumes: `make migrate`, `make reindex` (Tasks 3 e 5).

- [ ] **Step 1: Fotografia antes**

Run:

```bash
docker compose exec -T postgres psql -U postgres -d diario -Atc "select count(*) from acts; select count(*) from gazettes"
```

Expected: dois números; anotar. Em 22/09/2026 eram 73.526 atos e 1.740 edições (a base local só tem 2020 em diante).

- [ ] **Step 2: Migration**

Run: `make migrate`
Expected: log com `"applied":["005_act_location.sql"]`.

- [ ] **Step 3: Reindexar um dia e conferir**

Run:

```bash
make reindex FROM=2026-09-18 TO=2026-09-18
docker compose exec -T postgres psql -U postgres -d diario -Atc "select a.position, a.page_start, a.page_end, a.organ, left(a.title, 50) from acts a join gazettes g on g.id = a.gazette_id where g.published_at = '2026-09-18' order by a.position"
```

Expected: `"result":{"found":1,"reindexed":1,"failed":0}`; 51 atos, todos com página; órgãos entre `SEMAD`, `SEMED`, `SEMCI`, `SEMAS`, `SEOP`, `FMS`, `FUNASG`, `CONGES`, `SEMMATRAN` ou vazio no começo.

Conferência visual: abrir `https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf#page=N` para o primeiro, o último e três atos do meio e confirmar que o título está na página indicada.

- [ ] **Step 4: Reindexar tudo**

Run: `make reindex FROM=2020-01-01 TO=2026-12-31` (em background; leva vários minutos)
Expected: `failed` = 0. Se houver falhas, o log lista cada edição e o motivo; investigar antes de seguir.

- [ ] **Step 5: Conferir a base**

Run:

```bash
docker compose exec -T postgres psql -U postgres -d diario -Atc "
select count(*), count(*) filter (where page_start is null) from acts;
select organ, count(*) from acts group by 1 order by 2 desc limit 25;"
```

Expected: nenhum ato com página nula; total de atos comparado com o do Step 1 (anotar a diferença, se houver, e o motivo: o parser atual é o mesmo que indexou a base, então a diferença esperada é zero ou explicada pelas edições reindexadas no dia).

- [ ] **Step 6: Roadmap**

Em `docs/roadmap.md`, na Entrega 1, trocar o item de proveniência para registrar que a página é gravada e o de órgão para registrar que a sigla é gravada, com o commit; manter pendente o que é da 1b e da 1c (link, citação, filtro, nomes).

```markdown
- **Proveniência.** Guardar a página de cada ato ✅ (1a) e linkar
  `…pdf#page=N` (1b). ...
- **Órgão.** Persistir a sigla ✅ (1a) e expor filtro por secretaria (1c).
```

- [ ] **Step 7: Commit**

```bash
git add docs/roadmap.md
git commit -m "docs(roadmap): página e órgão por ato gravados (1a)"
```
