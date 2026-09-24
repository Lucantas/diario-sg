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
