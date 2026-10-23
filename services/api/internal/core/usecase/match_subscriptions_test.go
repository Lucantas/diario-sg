package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type matchActs struct {
	ports.ActRepository
	textQueries []string
	entityRefs  []domain.EntityRef
	filters     []domain.ActFilter
}

func (m *matchActs) Search(_ context.Context, f domain.ActFilter) ([]domain.ActHit, int, error) {
	m.filters = append(m.filters, f)
	return []domain.ActHit{{}}, 1, nil
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

type memLog struct {
	state     map[string]string
	leaseOver bool
	markErr   error
}

func newMemLog() *memLog { return &memLog{state: map[string]string{}} }

func (m *memLog) Claim(_ context.Context, sub, g string) (domain.NotificationClaim, error) {
	switch m.state[sub+g] {
	case "sent":
		return domain.ClaimAlreadySent, nil
	case "pending":
		if !m.leaseOver {
			return domain.ClaimInProgress, nil
		}
	}
	m.state[sub+g] = "pending"
	return domain.ClaimAcquired, nil
}

func (m *memLog) MarkSent(_ context.Context, sub, g string) error {
	if m.markErr != nil {
		return m.markErr
	}
	m.state[sub+g] = "sent"
	return nil
}

func (m *memLog) Release(_ context.Context, sub, g string) error {
	if m.state[sub+g] == "pending" {
		delete(m.state, sub+g)
	}
	return nil
}

func singleAlert() (*memGazettes, listSubs) {
	gazettes := newMemGazettes()
	gazettes.saved["g1"] = domain.Gazette{ID: "g1"}
	return gazettes, listSubs{active: []domain.Subscription{{ID: "s1", Query: "merenda"}}}
}

func TestRedeliveredEventAfterSuccessfulSendDoesNotSendAgain(t *testing.T) {
	gazettes, subs := singleAlert()
	notifier := &recNotifier{}
	uc := NewMatchSubscriptions(gazettes, &matchActs{}, subs, newMemLog(), notifier)

	for i := 0; i < 3; i++ {
		if err := uc.Execute(context.Background(), "g1"); err != nil {
			t.Fatal(err)
		}
	}

	if len(notifier.keys) != 1 || notifier.matches != 1 {
		t.Fatalf("esperava uma única chamada ao Notifier, vieram %v", notifier.keys)
	}
}

func TestFailureAfterSendIsRetriedWithTheSameIdempotencyKey(t *testing.T) {
	gazettes, subs := singleAlert()
	log, notifier := newMemLog(), &recNotifier{}
	log.markErr = errors.New("banco caiu depois do envio")
	uc := NewMatchSubscriptions(gazettes, &matchActs{}, subs, log, notifier)

	if err := uc.Execute(context.Background(), "g1"); err == nil {
		t.Fatal("falha ao registrar o envio deve devolver erro para o evento ser reentregue")
	}
	log.markErr = nil
	if err := uc.Execute(context.Background(), "g1"); !errors.Is(err, domain.ErrNotificationInProgress) {
		t.Fatalf("reentrega dentro do prazo da reserva deve pedir nova tentativa, veio %v", err)
	}
	log.leaseOver = true
	if err := uc.Execute(context.Background(), "g1"); err != nil {
		t.Fatal(err)
	}
	if err := uc.Execute(context.Background(), "g1"); err != nil {
		t.Fatal(err)
	}

	want := domain.AlertIdempotencyKey("s1", "g1")
	if len(notifier.keys) != 2 || notifier.keys[0] != want || notifier.keys[1] != want {
		t.Fatalf("as duas tentativas devem usar a chave %q: %v", want, notifier.keys)
	}
	if notifier.matches != 1 {
		t.Fatalf("esperava um e-mail com efeito, vieram %d", notifier.matches)
	}
}

func TestSendFailureReleasesTheAlertForTheRetry(t *testing.T) {
	gazettes, subs := singleAlert()
	notifier := &recNotifier{sendErr: errors.New("resend fora do ar")}
	uc := NewMatchSubscriptions(gazettes, &matchActs{}, subs, newMemLog(), notifier)

	if err := uc.Execute(context.Background(), "g1"); err == nil {
		t.Fatal("falha no envio deve devolver erro para o evento ser reentregue")
	}
	notifier.sendErr = nil
	if err := uc.Execute(context.Background(), "g1"); err != nil {
		t.Fatal(err)
	}

	if notifier.matches != 1 || len(notifier.keys) != 2 {
		t.Fatalf("o retry deve enviar: %d e-mails, chamadas %v", notifier.matches, notifier.keys)
	}
}

func TestAlertAlreadyAcceptedByTheProviderIsRecordedAsSent(t *testing.T) {
	gazettes, subs := singleAlert()
	log, notifier := newMemLog(), &recNotifier{sendErr: domain.ErrAlreadyNotified}
	uc := NewMatchSubscriptions(gazettes, &matchActs{}, subs, log, notifier)

	if err := uc.Execute(context.Background(), "g1"); err != nil {
		t.Fatal(err)
	}

	if log.state["s1g1"] != "sent" {
		t.Fatalf("alerta já aceito pelo provedor deve ficar como enviado: %q", log.state["s1g1"])
	}
}

func TestMatchSubscriptionsUsesEntityLinksForEntitySubscriptions(t *testing.T) {
	gazettes := newMemGazettes()
	gazettes.saved["g1"] = domain.Gazette{ID: "g1"}
	ref := domain.EntityRef{Kind: domain.EntityContrato, Key: "12/2024", Label: "12/2024"}
	subs := listSubs{active: []domain.Subscription{{ID: "s1", Query: "merenda OU lanche"}, {ID: "s2", Entity: &ref}}}
	acts, log, notifier := &matchActs{}, newMemLog(), &recNotifier{}
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

func TestMatchSubscriptionsSearchesTheEditionWithTheAlertFilter(t *testing.T) {
	gazettes := newMemGazettes()
	gazettes.saved["g1"] = domain.Gazette{ID: "g1"}
	filter := domain.AlertFilter{Type: domain.ActLicencaAmbiental, Theme: "meio_ambiente"}
	subs := listSubs{active: []domain.Subscription{{ID: "s1", Query: "loteamento OU condomínio", Filter: filter}}}
	acts, notifier := &matchActs{}, &recNotifier{}
	uc := NewMatchSubscriptions(gazettes, acts, subs, newMemLog(), notifier)

	if err := uc.Execute(context.Background(), "g1"); err != nil {
		t.Fatal(err)
	}

	if len(acts.textQueries) != 0 || len(acts.filters) != 1 {
		t.Fatalf("esperava só a busca filtrada: texto %v, filtros %v", acts.textQueries, acts.filters)
	}
	got := acts.filters[0]
	if got.GazetteID != "g1" || got.Type != domain.ActLicencaAmbiental || got.Theme != "meio_ambiente" || got.Query != "loteamento or condomínio" {
		t.Errorf("filtro inesperado: %+v", got)
	}
	if notifier.matches != 1 {
		t.Errorf("esperava 1 e-mail, veio %d", notifier.matches)
	}
}
