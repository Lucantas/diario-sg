package usecase

import (
	"context"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

// Subscriptions agrupa os casos de uso de inscrição em alertas.
type Subscriptions struct {
	repo     ports.SubscriptionRepository
	notifier ports.Notifier
	now      func() time.Time
}

func NewSubscriptions(r ports.SubscriptionRepository, n ports.Notifier) *Subscriptions {
	return &Subscriptions{repo: r, notifier: n, now: time.Now}
}

// Subscribe cria uma inscrição pendente e envia o e-mail de confirmação.
func (uc *Subscriptions) Subscribe(ctx context.Context, email, query string) (domain.Subscription, error) {
	s, err := domain.NewSubscription(email, query, uc.now())
	if err != nil {
		return domain.Subscription{}, err
	}
	if err := uc.repo.Create(ctx, &s); err != nil {
		return domain.Subscription{}, err
	}
	if err := uc.notifier.SendConfirmation(ctx, s); err != nil {
		return domain.Subscription{}, err
	}
	return s, nil
}

func (uc *Subscriptions) Confirm(ctx context.Context, token string) (domain.Subscription, error) {
	s, err := uc.repo.FindByConfirmToken(ctx, token)
	if err != nil {
		return domain.Subscription{}, err
	}
	if err := s.Confirm(uc.now()); err != nil {
		return domain.Subscription{}, err
	}
	return s, uc.repo.Update(ctx, s)
}

func (uc *Subscriptions) Unsubscribe(ctx context.Context, token string) error {
	s, err := uc.repo.FindByUnsubscribeToken(ctx, token)
	if err != nil {
		return err
	}
	s.Cancel()
	return uc.repo.Update(ctx, s)
}
