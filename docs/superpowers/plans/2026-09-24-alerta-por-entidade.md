# Alerta por entidade — plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** inscrição de alerta por e-mail e feed RSS por CNPJ, processo ou contrato, disparados pelas ligações de `entity_links`.

**Architecture:** `EntityRef` no domínio (tipo, chave normalizada, rótulo) é usado pela inscrição (`Subscription.Entity`) e pelo filtro da busca (`ActFilter.Entity`). O worker escolhe a busca por ligação (`EntityHitsInGazette`) ou a textual (`SearchInGazette`) conforme a inscrição. O front ganha `EntityAlert` nas páginas de empresa, processo e contrato.

**Tech Stack:** Go 1.x (`database/sql`, `lib/pq`, `html/template`), Postgres, React + Vite + Vitest.

**Spec:** `docs/superpowers/specs/2026-09-24-alerta-por-entidade-design.md`

## Global Constraints

- Regras de `AGENTS.md`: sem comentários explicativos no código, commits em português no formato conventional commits, sem link de sessão nos commits.
- Migrations já aplicadas não são editadas; a nova é `services/api/migrations/012_subscription_entity.sql`.
- Tipos de entidade do alerta: `cnpj`, `processo`, `contrato`.
- Parâmetro da API: `entity=<tipo>:<número>`.
- Trecho do e-mail: até 280 caracteres, marcação `⟦ ⟧`.
- Antes de dizer que terminou: `make lint test`, `make test-integration`, `cd apps/web && npm run typecheck && npm test`.

## Arquivos

| Arquivo | Papel |
| --- | --- |
| `services/api/internal/core/domain/entity_ref.go` (novo) | `EntityRef`, `ParseEntityRef`, `ParseEntityFilter`, `Description`, `MentionSnippet` |
| `services/api/internal/core/domain/subscription.go` | `Entity`, `NewEntitySubscription`, `Subject` |
| `services/api/internal/core/domain/act.go` | `ActFilter.Entity` |
| `services/api/migrations/012_subscription_entity.sql` (novo) | colunas da inscrição de entidade |
| `services/api/internal/adapters/postgres/subscriptions.go` | grava e lê as colunas novas |
| `services/api/internal/adapters/postgres/acts.go` | `EntityHitsInGazette` |
| `services/api/internal/adapters/postgres/filter.go` | filtro `Entity` |
| `services/api/internal/core/ports/ports.go` | `EntityHitsInGazette` na porta |
| `services/api/internal/core/usecase/subscriptions.go` | `SubscribeEntity` |
| `services/api/internal/core/usecase/match_subscriptions.go` | escolhe a busca pela inscrição |
| `services/api/internal/adapters/email/email.go` | `Subject()` e órgão |
| `services/api/internal/presentation/http/{router,dto,feed}.go` | pedido, resposta, filtro e feed |
| `apps/web/src/{api.ts,entity.ts,AlertForm.tsx,EntityAlert.tsx,EntityPage.tsx,CompanyPage.tsx,SearchPage.tsx,App.tsx}` | front |
| `README.md`, `docs/roadmap.md` | documentação |

---

### Task 1: `EntityRef` e trecho em volta da menção

**Files:**
- Create: `services/api/internal/core/domain/entity_ref.go`
- Test: `services/api/internal/core/domain/entity_ref_test.go`

**Interfaces:**
- Produces: `type EntityRef struct{ Kind EntityKind; Key, Label string }`; `ParseEntityRef(kind EntityKind, value string) (EntityRef, error)`; `ParseEntityFilter(s string) (*EntityRef, error)`; `(EntityRef) Description() string`; `MentionSnippet(body, evidence string) string`.

- [ ] **Step 1: testes que falham**

```go
package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestParseEntityRef(t *testing.T) {
	cases := []struct {
		kind  EntityKind
		value string
		want  EntityRef
	}{
		{EntityCNPJ, "12.345.678/0001-90", EntityRef{EntityCNPJ, "12345678000190", "12.345.678/0001-90"}},
		{EntityCNPJ, "12345678000190", EntityRef{EntityCNPJ, "12345678000190", "12.345.678/0001-90"}},
		{EntityProcesso, "8.189/2025", EntityRef{EntityProcesso, "81892025", "8.189/2025"}},
		{EntityProcesso, "Processo nº 8.189/2025", EntityRef{EntityProcesso, "81892025", "8.189/2025"}},
		{EntityContrato, "012/2024", EntityRef{EntityContrato, "12/2024", "12/2024"}},
		{EntityContrato, "30-fms-2011", EntityRef{EntityContrato, "30/FMS/2011", "30/FMS/2011"}},
	}
	for _, c := range cases {
		got, err := ParseEntityRef(c.kind, c.value)
		if err != nil || got != c.want {
			t.Errorf("ParseEntityRef(%s, %q) = %+v, %v; esperava %+v", c.kind, c.value, got, err, c.want)
		}
	}
}

func TestParseEntityRefRejectsInvalidInput(t *testing.T) {
	for _, c := range []struct {
		kind  EntityKind
		value string
	}{{EntityValor, "100"}, {"bobagem", "1/2024"}, {EntityCNPJ, "123"}, {EntityProcesso, "12"}, {EntityContrato, "abc"}} {
		if _, err := ParseEntityRef(c.kind, c.value); err == nil {
			t.Errorf("ParseEntityRef(%s, %q) deveria falhar", c.kind, c.value)
		}
	}
}

func TestParseEntityFilter(t *testing.T) {
	ref, err := ParseEntityFilter("contrato:012/2024")
	if err != nil || ref == nil || ref.Key != "12/2024" {
		t.Fatalf("veio %+v %v", ref, err)
	}
	if ref, err := ParseEntityFilter(""); ref != nil || err != nil {
		t.Errorf("filtro vazio deveria ser nil, veio %+v %v", ref, err)
	}
	for _, s := range []string{"cnpj", "cnpj:123", "valor:100"} {
		if _, err := ParseEntityFilter(s); !errors.Is(err, ErrInvalidFilter) {
			t.Errorf("%q deveria dar ErrInvalidFilter, veio %v", s, err)
		}
	}
}

func TestEntityRefDescription(t *testing.T) {
	for ref, want := range map[EntityRef]string{
		{EntityCNPJ, "12345678000190", "12.345.678/0001-90"}: "CNPJ 12.345.678/0001-90",
		{EntityProcesso, "81892025", "8.189/2025"}:           "processo 8.189/2025",
		{EntityContrato, "12/2024", "12/2024"}:               "contrato 12/2024",
	} {
		if got := ref.Description(); got != want {
			t.Errorf("veio %q, esperava %q", got, want)
		}
	}
}

func TestMentionSnippetCentersTheEvidence(t *testing.T) {
	body := strings.Repeat("antes ", 60) + "CNPJ: 12.345.678/0001-90." + strings.Repeat(" depois", 60)

	got := MentionSnippet(body, "12.345.678/0001-90")

	if !strings.Contains(got, "⟦12.345.678/0001-90⟧") || !strings.HasPrefix(got, "antes") {
		t.Fatalf("trecho inesperado: %q", got)
	}
	if n := len([]rune(got)) - 2; n > 280 {
		t.Errorf("trecho com %d caracteres, esperava até 280", n)
	}
}

func TestMentionSnippetCollapsesWhitespaceAndKeepsAccents(t *testing.T) {
	got := MentionSnippet("Contratação  da\nempresa, contrato nº\n55/2026.", "nº 55/2026")

	if got != "Contratação da empresa, contrato ⟦nº 55/2026⟧." {
		t.Errorf("veio %q", got)
	}
}

func TestMentionSnippetWithoutEvidenceIsTheStartOfTheBody(t *testing.T) {
	body := strings.Repeat("a", 400)

	if got := MentionSnippet(body, "xyz"); got != strings.Repeat("a", 280) {
		t.Errorf("veio %d caracteres", len(got))
	}
	if got := MentionSnippet("curto", ""); got != "curto" {
		t.Errorf("veio %q", got)
	}
}
```

- [ ] **Step 2:** `cd services/api && go test ./internal/core/domain/ -run 'EntityRef|EntityFilter|MentionSnippet'` → não compila (`undefined: ParseEntityRef`).

- [ ] **Step 3: implementação**

```go
package domain

import (
	"strings"
	"unicode/utf8"
)

const (
	mentionSnippetRunes = 280
	mentionLeadRunes    = 120
)

type EntityRef struct {
	Kind  EntityKind
	Key   string
	Label string
}

var entityDescriptionPrefix = map[EntityKind]string{
	EntityCNPJ:     "CNPJ",
	EntityProcesso: "processo",
	EntityContrato: "contrato",
}

func ParseEntityRef(kind EntityKind, value string) (EntityRef, error) {
	if !IsLinkedKind(kind) {
		return EntityRef{}, ErrInvalidInput
	}
	key, err := ParseEntityInput(kind, value)
	if err != nil {
		return EntityRef{}, err
	}
	return EntityRef{Kind: kind, Key: key, Label: entityRefLabel(kind, key, value)}, nil
}

func entityRefLabel(kind EntityKind, key, value string) string {
	switch kind {
	case EntityCNPJ:
		return FormatCNPJ(key)
	case EntityContrato:
		return key
	}
	return EntityLabel(kind, value)
}

func ParseEntityFilter(s string) (*EntityRef, error) {
	if s == "" {
		return nil, nil
	}
	kind, value, ok := strings.Cut(s, ":")
	if !ok {
		return nil, ErrInvalidFilter
	}
	ref, err := ParseEntityRef(EntityKind(kind), value)
	if err != nil {
		return nil, ErrInvalidFilter
	}
	return &ref, nil
}

func (r EntityRef) Description() string {
	return entityDescriptionPrefix[r.Kind] + " " + r.Label
}

func MentionSnippet(body, evidence string) string {
	text := strings.Join(strings.Fields(body), " ")
	needle := strings.Join(strings.Fields(evidence), " ")
	at := strings.Index(text, needle)
	runes := []rune(text)
	if needle == "" || at < 0 {
		return string(runes[:min(len(runes), mentionSnippetRunes)])
	}
	start := utf8.RuneCountInString(text[:at])
	stop := start + utf8.RuneCountInString(needle)
	from := max(0, start-mentionLeadRunes)
	to := max(stop, min(len(runes), from+mentionSnippetRunes))
	return string(runes[from:start]) + "⟦" + string(runes[start:stop]) + "⟧" + string(runes[stop:to])
}
```

- [ ] **Step 4:** mesmo comando → PASS.
- [ ] **Step 5:** `git commit -m "feat(api): referência de entidade e trecho em volta da menção"`

### Task 2: inscrição de entidade (domínio, migration, repositório, caso de uso)

**Files:**
- Modify: `services/api/internal/core/domain/subscription.go`, `services/api/internal/adapters/postgres/subscriptions.go`, `services/api/internal/core/usecase/subscriptions.go`
- Create: `services/api/migrations/012_subscription_entity.sql`
- Test: `services/api/internal/core/domain/subscription_test.go`, `services/api/internal/integration/entity_alert_test.go`

**Interfaces:**
- Consumes: `EntityRef`, `ParseEntityRef` (Task 1).
- Produces: `Subscription.Entity *EntityRef`; `NewEntitySubscription(email string, ref EntityRef, now time.Time) (Subscription, error)`; `(Subscription) Subject() string`; `(*Subscriptions) SubscribeEntity(ctx, email string, kind domain.EntityKind, value string) (domain.Subscription, error)`.

- [ ] **Step 1: testes de domínio que falham**

```go
func TestNewEntitySubscription(t *testing.T) {
	ref := EntityRef{Kind: EntityCNPJ, Key: "12345678000190", Label: "12.345.678/0001-90"}

	s, err := NewEntitySubscription(" A@B.com ", ref, time.Now())

	if err != nil || s.Email != "a@b.com" || s.Query != "" || s.Entity == nil || *s.Entity != ref || s.Status != SubscriptionPending {
		t.Fatalf("inscrição inesperada: %+v %v", s, err)
	}
	if s.ConfirmToken == "" || s.ConfirmToken == s.UnsubscribeToken {
		t.Error("tokens devem existir e ser diferentes")
	}
	if _, err := NewEntitySubscription("x", ref, time.Now()); !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("esperava ErrInvalidEmail, veio %v", err)
	}
}

func TestSubscriptionSubject(t *testing.T) {
	termo := Subscription{Query: "merenda escolar"}
	entidade := Subscription{Entity: &EntityRef{Kind: EntityProcesso, Key: "81892025", Label: "8.189/2025"}}

	if got := termo.Subject(); got != "“merenda escolar”" {
		t.Errorf("termo: %q", got)
	}
	if got := entidade.Subject(); got != "processo 8.189/2025" {
		t.Errorf("entidade: %q", got)
	}
}
```

- [ ] **Step 2:** `go test ./internal/core/domain/ -run Subscription` → não compila.

- [ ] **Step 3: domínio**

`NewSubscription` passa a usar um construtor comum:

```go
type Subscription struct {
	ID               string
	Email            string
	Query            string
	Entity           *EntityRef
	Status           SubscriptionStatus
	ConfirmToken     string
	UnsubscribeToken string
	CreatedAt        time.Time
	ConfirmedAt      *time.Time
}

func NewSubscription(email, query string, now time.Time) (Subscription, error) {
	s, err := newPendingSubscription(email, now)
	if err != nil {
		return Subscription{}, err
	}
	s.Query = strings.Join(strings.Fields(query), " ")
	if n := utf8.RuneCountInString(s.Query); n < 3 || n > 200 {
		return Subscription{}, ErrInvalidQuery
	}
	return s, nil
}

func NewEntitySubscription(email string, ref EntityRef, now time.Time) (Subscription, error) {
	s, err := newPendingSubscription(email, now)
	if err != nil {
		return Subscription{}, err
	}
	s.Entity = &ref
	return s, nil
}

func (s Subscription) Subject() string {
	if s.Entity != nil {
		return s.Entity.Description()
	}
	return "“" + s.Query + "”"
}

func newPendingSubscription(email string, now time.Time) (Subscription, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return Subscription{}, ErrInvalidEmail
	}
	confirm, err := newToken()
	if err != nil {
		return Subscription{}, err
	}
	unsub, err := newToken()
	if err != nil {
		return Subscription{}, err
	}
	return Subscription{Email: email, Status: SubscriptionPending, ConfirmToken: confirm,
		UnsubscribeToken: unsub, CreatedAt: now}, nil
}
```

(O e-mail continua sendo validado antes da consulta, como no `NewSubscription` atual.)

- [ ] **Step 4:** `go test ./internal/core/domain/` → PASS.

- [ ] **Step 5: migration**

```sql
ALTER TABLE subscriptions
    ALTER COLUMN query DROP NOT NULL,
    ADD COLUMN entity_kind  text CHECK (entity_kind IN ('cnpj', 'processo', 'contrato')),
    ADD COLUMN entity_key   text,
    ADD COLUMN entity_label text,
    ADD CONSTRAINT subscriptions_query_or_entity CHECK (
        (query IS NOT NULL AND entity_kind IS NULL AND entity_key IS NULL AND entity_label IS NULL)
        OR (query IS NULL AND entity_kind IS NOT NULL AND entity_key IS NOT NULL AND entity_label IS NOT NULL)
    );
```

- [ ] **Step 6: repositório**

```go
const subCols = `id, email, coalesce(query, ''), entity_kind, entity_key, entity_label, status, confirm_token, unsubscribe_token, created_at, confirmed_at`

func (r *SubscriptionRepo) Create(ctx context.Context, s *domain.Subscription) error {
	var query, kind, key, label sql.NullString
	if s.Entity != nil {
		kind = sql.NullString{String: string(s.Entity.Kind), Valid: true}
		key = sql.NullString{String: s.Entity.Key, Valid: true}
		label = sql.NullString{String: s.Entity.Label, Valid: true}
	} else {
		query = sql.NullString{String: s.Query, Valid: true}
	}
	return r.db.QueryRowContext(ctx, `
		INSERT INTO subscriptions (email, query, entity_kind, entity_key, entity_label, status, confirm_token, unsubscribe_token, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		s.Email, query, kind, key, label, string(s.Status), s.ConfirmToken, s.UnsubscribeToken, s.CreatedAt,
	).Scan(&s.ID)
}

func scanSub(row scanner) (domain.Subscription, error) {
	var s domain.Subscription
	var status string
	var kind, key, label sql.NullString
	var confirmed sql.NullTime
	err := row.Scan(&s.ID, &s.Email, &s.Query, &kind, &key, &label, &status, &s.ConfirmToken, &s.UnsubscribeToken, &s.CreatedAt, &confirmed)
	s.Status = domain.SubscriptionStatus(status)
	if kind.Valid {
		s.Entity = &domain.EntityRef{Kind: domain.EntityKind(kind.String), Key: key.String, Label: label.String}
	}
	if confirmed.Valid {
		s.ConfirmedAt = &confirmed.Time
	}
	return s, err
}
```

- [ ] **Step 7: caso de uso**

```go
func (uc *Subscriptions) Subscribe(ctx context.Context, email, query string) (domain.Subscription, error) {
	s, err := domain.NewSubscription(email, query, uc.now())
	if err != nil {
		return domain.Subscription{}, err
	}
	return uc.create(ctx, s)
}

func (uc *Subscriptions) SubscribeEntity(ctx context.Context, email string, kind domain.EntityKind, value string) (domain.Subscription, error) {
	ref, err := domain.ParseEntityRef(kind, value)
	if err != nil {
		return domain.Subscription{}, err
	}
	s, err := domain.NewEntitySubscription(email, ref, uc.now())
	if err != nil {
		return domain.Subscription{}, err
	}
	return uc.create(ctx, s)
}

func (uc *Subscriptions) create(ctx context.Context, s domain.Subscription) (domain.Subscription, error) {
	if err := uc.repo.Create(ctx, &s); err != nil {
		return domain.Subscription{}, err
	}
	if err := uc.notifier.SendConfirmation(ctx, s); err != nil {
		return domain.Subscription{}, err
	}
	return s, nil
}
```

- [ ] **Step 8: teste de integração** (`entity_alert_test.go`, build tag `integration`): cria inscrição de entidade pelo `SubscriptionRepo`, lê por `FindByConfirmToken` e confere `Entity` e `Query == ""`; confirma com `Update` e confere que `ListActive` a devolve; inserir linha com `query` e `entity_kind` juntos pelo SQL dá erro de `CHECK`.

- [ ] **Step 9:** `make test` e `make test-integration` → PASS. Commit: `feat(api): inscrição de alerta por CNPJ, processo ou contrato`.

### Task 3: disparo do alerta de entidade no worker

**Files:**
- Modify: `services/api/internal/core/ports/ports.go`, `services/api/internal/adapters/postgres/acts.go`, `services/api/internal/core/usecase/match_subscriptions.go`
- Test: `services/api/internal/core/usecase/match_subscriptions_test.go` (novo), `services/api/internal/integration/entity_alert_test.go`

**Interfaces:**
- Consumes: `EntityRef`, `MentionSnippet` (Task 1); `Subscription.Entity` (Task 2).
- Produces: `ActRepository.EntityHitsInGazette(ctx context.Context, gazetteID string, ref domain.EntityRef) ([]domain.ActHit, error)`.

- [ ] **Step 1: teste unitário que falha** (fakes no próprio arquivo, embutindo as interfaces para só implementar o necessário):

```go
package usecase

import (
	"context"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type matchActs struct {
	ports.ActRepository
	textQueries []string
	entityRefs  []domain.EntityRef
}

func (m *matchActs) SearchInGazette(_ context.Context, _ string, q string) ([]domain.ActHit, error) {
	m.textQueries = append(m.textQueries, q)
	return []domain.ActHit{{}}, nil
}

func (m *matchActs) EntityHitsInGazette(_ context.Context, _ string, ref domain.EntityRef) ([]domain.ActHit, error) {
	m.entityRefs = append(m.entityRefs, ref)
	return []domain.ActHit{{}}, nil
}

type listSubs struct {
	ports.SubscriptionRepository
	active []domain.Subscription
}

func (l listSubs) ListActive(context.Context) ([]domain.Subscription, error) { return l.active, nil }

type memLog struct{ sent map[string]bool }

func (m *memLog) WasSent(_ context.Context, sub, g string) (bool, error) { return m.sent[sub+g], nil }
func (m *memLog) MarkSent(_ context.Context, sub, g string) error {
	m.sent[sub+g] = true
	return nil
}

func TestMatchSubscriptionsUsesEntityLinksForEntitySubscriptions(t *testing.T) {
	gazettes := newMemGazettes()
	gazettes.saved["g1"] = domain.Gazette{ID: "g1"}
	ref := domain.EntityRef{Kind: domain.EntityContrato, Key: "12/2024", Label: "12/2024"}
	subs := listSubs{active: []domain.Subscription{{ID: "s1", Query: "merenda OU lanche"}, {ID: "s2", Entity: &ref}}}
	acts, log, notifier := &matchActs{}, &memLog{sent: map[string]bool{}}, &recNotifier{}
	uc := NewMatchSubscriptions(gazettes, acts, subs, log, notifier)

	for i := 0; i < 2; i++ {
		if err := uc.Execute(context.Background(), "g1"); err != nil {
			t.Fatal(err)
		}
	}

	if len(acts.textQueries) != 1 || acts.textQueries[0] != "merenda or lanche" {
		t.Errorf("busca textual inesperada: %v", acts.textQueries)
	}
	if len(acts.entityRefs) != 1 || acts.entityRefs[0] != ref {
		t.Errorf("busca por entidade inesperada: %v", acts.entityRefs)
	}
	if notifier.matches != 2 {
		t.Errorf("esperava 2 e-mails, veio %d", notifier.matches)
	}
}
```

- [ ] **Step 2:** `go test ./internal/core/usecase/ -run MatchSubscriptions` → não compila (`EntityHitsInGazette` não existe na porta).

- [ ] **Step 3: porta, caso de uso e Postgres**

Em `ports.ActRepository`, ao lado de `SearchInGazette`:

```go
	EntityHitsInGazette(ctx context.Context, gazetteID string, ref domain.EntityRef) ([]domain.ActHit, error)
```

Em `MatchSubscriptions.Execute`, trocar a chamada de `SearchInGazette` por `uc.hitsFor(ctx, s, g.ID)`:

```go
func (uc *MatchSubscriptions) hitsFor(ctx context.Context, s domain.Subscription, gazetteID string) ([]domain.ActHit, error) {
	if s.Entity != nil {
		return uc.acts.EntityHitsInGazette(ctx, gazetteID, *s.Entity)
	}
	return uc.acts.SearchInGazette(ctx, gazetteID, domain.TranslateOperators(s.Query))
}
```

Em `postgres/acts.go`:

```go
func (r *ActRepo) EntityHitsInGazette(ctx context.Context, gazetteID string, ref domain.EntityRef) ([]domain.ActHit, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, a.organ, a.body,
		       g.edition_number, g.published_at, g.is_extra, g.source_url, g.source, min(l.evidence), `+cnpjsSubquery+`
		FROM entities e
		JOIN entity_links l ON l.entity_id = e.id AND l.record_kind = $4
		JOIN acts a ON a.id = l.record_id::uuid
		JOIN gazettes g ON g.id = a.gazette_id
		WHERE e.kind = $2 AND e.key = $3 AND a.gazette_id = $1
		GROUP BY a.id, g.id
		ORDER BY a.position
		LIMIT 20`, gazetteID, string(ref.Kind), ref.Key, domain.RecordAct)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []domain.ActHit
	for rows.Next() {
		var h domain.ActHit
		var typ, body, evidence string
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Position, &h.Organ, &body, &h.EditionNumber,
			&h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Source, &evidence, pq.Array(&h.CNPJs)); err != nil {
			return nil, err
		}
		h.Type = domain.ActType(typ)
		h.Snippet = domain.MentionSnippet(body, evidence)
		hits = append(hits, h)
	}
	return hits, rows.Err()
}
```

Qualquer outro fake de `ActRepository` que não embuta a interface ganha o método (o compilador aponta).

- [ ] **Step 4: integração** em `entity_alert_test.go`: indexa com `newServerFor` um texto com dois contratos (`EXTRATO DO CONTRATO Nº 012/2024 … CNPJ 12.345.678/0001-90` e `EXTRATO DO CONTRATO Nº 13/2024`); `EntityHitsInGazette` com `contrato 12/2024` traz só o primeiro, com `⟦` no trecho; com CNPJ traz o mesmo ato; com `contrato 99/2024` traz nada. Roda `MatchSubscriptions` com uma inscrição ativa de CNPJ e um `email.Notifier` sobre a caixa `inbox`: um e-mail com assunto `CNPJ 12.345.678/0001-90 no Diário da Prefeitura de 18/09`; rodar de novo não manda outro.

- [ ] **Step 5:** `make test` e `make test-integration` → PASS. Commit: `feat(api): alerta de entidade dispara pelas ligações da edição`.

### Task 4: e-mails com o assunto da inscrição

**Files:**
- Modify: `services/api/internal/adapters/email/email.go`
- Test: `services/api/internal/adapters/email/email_test.go`

**Interfaces:**
- Consumes: `Subscription.Subject()` (Task 2).

- [ ] **Step 1: teste que falha**

```go
func TestEntityAlertEmails(t *testing.T) {
	sender := &recSender{}
	n := NewNotifier(sender, "https://site")
	sub := domain.Subscription{Email: "a@b.c", ConfirmToken: "c", UnsubscribeToken: "u",
		Entity: &domain.EntityRef{Kind: domain.EntityProcesso, Key: "81892025", Label: "8.189/2025"}}
	hits := []domain.ActHit{{Act: domain.Act{Title: "EXTRATO DO CONTRATO", Organ: "SEMED"}, Snippet: "processo ⟦8.189/2025⟧"}}
	g := domain.Gazette{EditionNumber: "1771", PublishedAt: time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)}

	if err := n.SendConfirmation(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if err := n.SendMatches(context.Background(), sub, g, hits); err != nil {
		t.Fatal(err)
	}

	confirm, matches := sender.sent[0], sender.sent[1]
	if !strings.Contains(confirm.HTML, "<strong>processo 8.189/2025</strong>") {
		t.Errorf("confirmação: %s", confirm.HTML)
	}
	if matches.Subject != "Processo 8.189/2025 no Diário da Prefeitura de 03/11" ||
		!strings.Contains(matches.HTML, "para <strong>processo 8.189/2025</strong>") ||
		!strings.Contains(matches.HTML, "EXTRATO DO CONTRATO</strong> (SEMED)") {
		t.Errorf("alerta: %q\n%s", matches.Subject, matches.HTML)
	}
}
```

- [ ] **Step 2:** `go test ./internal/adapters/email/` → FAIL.

- [ ] **Step 3: implementação**: nos dois templates, `{{.Query}}` vira `{{.Subject}}` e o mapa passa `"Subject": s.Subject()`; no de resultados, `<strong>{{.Title}}</strong>{{if .Organ}} ({{.Organ}}){{end}}`; o assunto vira `fmt.Sprintf("%s no Diário da %s de %s", upperFirst(s.Subject()), …)` com

```go
func upperFirst(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}
```

- [ ] **Step 4:** `go test ./internal/adapters/email/` → PASS (o teste antigo, com `“cota parlamentar”`, continua passando).
- [ ] **Step 5:** commit `feat(email): alerta de entidade no assunto e órgão de cada ato`.

### Task 5: HTTP — inscrição, filtro `entity` e feed de entidade

**Files:**
- Modify: `services/api/internal/core/domain/act.go`, `services/api/internal/adapters/postgres/filter.go`, `services/api/internal/presentation/http/{router,dto,feed}.go`
- Test: `services/api/internal/presentation/http/feed_test.go`, `services/api/internal/integration/entity_alert_test.go`

**Interfaces:**
- Consumes: `ParseEntityFilter`, `EntityRef.Description`, `EntitySlug` (domínio); `SubscribeEntity` (Task 2).
- Produces: `ActFilter.Entity *EntityRef`; JSON `POST /v1/subscriptions` `{"email","entity":{"kind","value"}}`; resposta `{"query","subject","entity":{"kind","key","label"}|null,"status"}`.

- [ ] **Step 1: testes unitários que falham** (em `feed_test.go`):

```go
func TestSiteSearchURLOfEntityIsTheEntityPage(t *testing.T) {
	cases := map[domain.EntityRef]string{
		{Kind: domain.EntityCNPJ, Key: "12345678000190", Label: "12.345.678/0001-90"}: "https://site/empresa/12345678000190",
		{Kind: domain.EntityContrato, Key: "30/FMS/2011", Label: "30/FMS/2011"}:        "https://site/contrato/30-FMS-2011",
		{Kind: domain.EntityProcesso, Key: "81892025", Label: "8.189/2025"}:            "https://site/processo/8.189-2025",
	}
	for ref, want := range cases {
		if got := siteSearchURL("https://site/", domain.ActFilter{Entity: &ref}); got != want {
			t.Errorf("veio %s, esperava %s", got, want)
		}
	}
}

func TestFeedTitle(t *testing.T) {
	ref := domain.EntityRef{Kind: domain.EntityCNPJ, Key: "12345678000190", Label: "12.345.678/0001-90"}
	for f, want := range map[*domain.ActFilter]string{
		{}:                  "Diário SG: atos publicados",
		{Query: "merenda"}:  "Diário SG: merenda",
		{Entity: &ref}:      "Diário SG: CNPJ 12.345.678/0001-90",
	} {
		if got := feedTitle(*f); got != want {
			t.Errorf("veio %q, esperava %q", got, want)
		}
	}
}
```

- [ ] **Step 2:** `go test ./internal/presentation/http/` → não compila.

- [ ] **Step 3: implementação**

- `ActFilter`: campo `Entity *EntityRef` depois de `Name`.
- `filterFromQuery`, antes do `return f, true`:

```go
	if f.Entity, err = domain.ParseEntityFilter(q.Get("entity")); err != nil {
		writeError(w, err, a.Log)
		return f, false
	}
```

- `filterSQL`, antes do filtro de valor:

```go
	if f.Entity != nil {
		b.WriteString(" AND a.id IN (SELECT l.record_id::uuid FROM entities e JOIN entity_links l ON l.entity_id = e.id AND l.record_kind = '" +
			domain.RecordAct + "' WHERE e.kind = " + param(string(f.Entity.Kind)) + " AND e.key = " + param(f.Entity.Key) + ")")
	}
```

- `feed.go`: o título sai de `feedTitle(f)` (usa `f.Query`, que o handler já tem normalizado igual ao `q` cru para termos sem `OU`; o teste de integração existente com `q=limpeza` confirma):

```go
func feedTitle(f domain.ActFilter) string {
	switch {
	case f.Entity != nil:
		return "Diário SG: " + f.Entity.Description()
	case strings.TrimSpace(f.Query) != "":
		return "Diário SG: " + strings.TrimSpace(f.Query)
	}
	return "Diário SG: atos publicados"
}
```

  Como o `ActFeed` normaliza uma cópia, o handler passa o `f` cru (antes da normalização) para `feedTitle`, como hoje lê `r.URL.Query().Get("q")`. `siteSearchURL` começa com:

```go
	if f.Entity != nil {
		return entityPageURL(base, *f.Entity)
	}
```

```go
func entityPageURL(base string, ref domain.EntityRef) string {
	base = strings.TrimRight(base, "/")
	if ref.Kind == domain.EntityCNPJ {
		return base + "/empresa/" + ref.Key
	}
	return base + "/" + string(ref.Kind) + "/" + url.PathEscape(domain.EntitySlug(ref.Label))
}
```

- `dto.go`:

```go
type subscribeRequest struct {
	Email  string       `json:"email"`
	Query  string       `json:"query"`
	Entity *entityInput `json:"entity"`
}

type entityInput struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type subscriptionEntityDTO struct {
	Kind  string `json:"kind"`
	Key   string `json:"key"`
	Label string `json:"label"`
}

type subscriptionDTO struct {
	Query   string                 `json:"query"`
	Subject string                 `json:"subject"`
	Entity  *subscriptionEntityDTO `json:"entity"`
	Status  string                 `json:"status"`
}

func toSubscriptionDTO(s domain.Subscription) subscriptionDTO {
	dto := subscriptionDTO{Query: s.Query, Subject: s.Subject(), Status: string(s.Status)}
	if s.Entity != nil {
		dto.Entity = &subscriptionEntityDTO{Kind: string(s.Entity.Kind), Key: s.Entity.Key, Label: s.Entity.Label}
	}
	return dto
}
```

- `router.go`, `subscribe`:

```go
	var s domain.Subscription
	var err error
	switch {
	case req.Entity != nil && req.Query != "":
		err = domain.ErrInvalidInput
	case req.Entity != nil:
		s, err = a.Subscriptions.SubscribeEntity(r.Context(), req.Email, domain.EntityKind(req.Entity.Kind), req.Entity.Value)
	default:
		s, err = a.Subscriptions.Subscribe(r.Context(), req.Email, req.Query)
	}
	if err != nil {
		writeError(w, err, a.Log)
		return
	}
	writeJSON(w, http.StatusAccepted, toSubscriptionDTO(s))
```

  e `confirm` responde `toSubscriptionDTO(s)`.

- [ ] **Step 4: integração** em `entity_alert_test.go`, com o texto da Task 3: `GET /v1/acts?entity=contrato:12-2024` traz 1 ato; `entity=cnpj:12.345.678/0001-90` traz 1; `entity=contrato:abc` é 400; o feed `?entity=cnpj:12345678000190` tem título `Diário SG: CNPJ 12.345.678/0001-90`, link `https://web.exemplo/empresa/12345678000190` e 1 item; `POST /v1/subscriptions` com `{"email","entity":{"kind":"cnpj","value":"12.345.678/0001-90"}}` é 202 com `subject` e `entity.key`; com `query` e `entity` é 400; com `entity.kind` `valor` é 400. O servidor precisa de `Subscriptions`: `newServerFor` ganha `Subscriptions: usecase.NewSubscriptions(postgres.NewSubscriptionRepo(db), email.NewNotifier(&inbox{}, "https://web.exemplo"))`.

- [ ] **Step 5:** `make lint test` e `make test-integration` → PASS. Commit: `feat(api): filtro entity na busca e no RSS e inscrição de entidade na API`.

### Task 6: front — alerta e RSS nas páginas de entidade

**Files:**
- Create: `apps/web/src/AlertForm.tsx`, `apps/web/src/EntityAlert.tsx`
- Modify: `apps/web/src/api.ts`, `apps/web/src/entity.ts`, `apps/web/src/SearchPage.tsx`, `apps/web/src/EntityPage.tsx`, `apps/web/src/CompanyPage.tsx`, `apps/web/src/App.tsx`
- Test: `apps/web/src/entity.test.ts`

**Interfaces:**
- Consumes: API da Task 5.
- Produces: `AlertEntityKind = "cnpj" | "processo" | "contrato"`; `subscribeEntity(email, kind, value)`; `entityFeedUrl(kind, value): string`; `alertSubjectLabel(kind, label): string`.

- [ ] **Step 1: testes que falham** em `entity.test.ts`:

```ts
describe("entityFeedUrl", () => {
  it("leva o tipo e o número no parâmetro entity", () => {
    expect(entityFeedUrl("contrato", "30/FMS/2011")).toBe("/api/v1/feeds/acts?entity=contrato%3A30%2FFMS%2F2011");
  });
});

describe("alertSubjectLabel", () => {
  it("nomeia a entidade como no e-mail", () => {
    expect(alertSubjectLabel("cnpj", "12.345.678/0001-90")).toBe("o CNPJ 12.345.678/0001-90");
    expect(alertSubjectLabel("processo", "8.189/2025")).toBe("o processo 8.189/2025");
  });
});
```

- [ ] **Step 2:** `cd apps/web && npm test` → FAIL.

- [ ] **Step 3: implementação**

`entity.ts`:

```ts
export type AlertEntityKind = EntityKind | "cnpj";

const ALERT_PREFIX: Record<AlertEntityKind, string> = { cnpj: "o CNPJ", processo: "o processo", contrato: "o contrato" };

export function entityFeedUrl(kind: AlertEntityKind, value: string) {
  return `/api/v1/feeds/acts?${new URLSearchParams({ entity: `${kind}:${value}` })}`;
}

export function alertSubjectLabel(kind: AlertEntityKind, label: string) {
  return `${ALERT_PREFIX[kind]} ${label}`;
}
```

(`URLSearchParams` codifica `:` como `%3A` e `/` como `%2F`.)

`api.ts`: `subscribe` e `confirmSubscription` devolvem `Subscription { query: string; subject: string; status: string }`; novo

```ts
export function subscribeEntity(email: string, kind: AlertEntityKind, value: string) {
  return request<Subscription>("/v1/subscriptions", {
    method: "POST",
    body: JSON.stringify({ email, entity: { kind, value } }),
  });
}
```

`AlertForm.tsx`: o `AlertForm` de `SearchPage.tsx` sai para cá, com props `{ title: string; description: string; onSubscribe: (email: string) => Promise<unknown> }`; `SearchPage` passa `title={`Avisar quando “${state.q}” aparecer de novo`}`, a descrição atual e `onSubscribe={(email) => subscribe(email, state.q)}`.

`EntityAlert.tsx`:

```tsx
import { subscribeEntity } from "./api";
import { AlertForm } from "./AlertForm";
import { AlertEntityKind, alertSubjectLabel, entityFeedUrl } from "./entity";

export function EntityAlert({ kind, value, label }: { kind: AlertEntityKind; value: string; label: string }) {
  const subject = alertSubjectLabel(kind, label);
  return (
    <>
      <AlertForm
        title={`Avisar quando ${subject} aparecer de novo`}
        description="Você recebe um e-mail no dia em que uma nova edição citar este número."
        onSubscribe={(email) => subscribeEntity(email, kind, value)}
      />
      <p className="feed">
        Prefere RSS? <a href={window.location.origin + entityFeedUrl(kind, value)}>Assine o feed de {subject}</a>: os 50
        atos mais recentes que citam este número, sem precisar de e-mail.
      </p>
    </>
  );
}
```

`EntityPage.tsx`: `<EntityAlert kind={kind} value={data.label} label={data.label} />` depois de "Citados junto". `CompanyPage.tsx`: `<EntityAlert kind="cnpj" value={cnpj} label={formatCnpj(cnpj)} />` depois da linha do tempo. `App.tsx`: `Alerta confirmado. Você será avisado quando houver ato novo sobre ${s.subject}.`

- [ ] **Step 4:** `npm test` e `npm run typecheck` → PASS.
- [ ] **Step 5:** commit `feat(web): alerta e RSS nas páginas de empresa, processo e contrato`.

### Task 7: documentação

**Files:** `README.md`, `docs/roadmap.md`

- [ ] README, tabela da API: `/v1/acts` ganha `entity=<tipo>:<número>` (`cnpj`, `processo` ou `contrato`; atos ligados ao número, como nas páginas de entidade; inválido é 400), válido também na exportação e no feed; `POST /v1/subscriptions` aceita `{"email","entity":{"kind","value"}}` em vez de `query`, e a resposta traz `subject` e `entity`.
- [ ] Roadmap, Entrega 2: "Alerta por entidade" com ✅, o que foi feito, os commits e o link para a spec; nas pendências de nuvem, lembrar que a migration 012 precisa ir antes da imagem nova da API e do worker.
- [ ] Commit `docs(roadmap): alerta por entidade`.
