package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type MatchSubscriptions struct {
	gazettes ports.GazetteRepository
	acts     ports.ActRepository
	subs     ports.SubscriptionRepository
	log      ports.NotificationLog
	notifier ports.Notifier
}

func NewMatchSubscriptions(g ports.GazetteRepository, a ports.ActRepository, s ports.SubscriptionRepository, l ports.NotificationLog, n ports.Notifier) *MatchSubscriptions {
	return &MatchSubscriptions{gazettes: g, acts: a, subs: s, log: l, notifier: n}
}

func (uc *MatchSubscriptions) Execute(ctx context.Context, gazetteID string) error {
	g, err := uc.gazettes.FindByID(ctx, gazetteID)
	if err != nil {
		return err
	}
	subs, err := uc.subs.ListActive(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for _, s := range subs {
		if err := uc.notify(ctx, s, g); err != nil {
			errs = append(errs, fmt.Errorf("alerta %s: %w", s.ID, err))
		}
	}

	return errors.Join(errs...)
}

func (uc *MatchSubscriptions) notify(ctx context.Context, s domain.Subscription, g domain.Gazette) error {
	claim, err := uc.log.Claim(ctx, s.ID, g.ID)
	if err != nil {
		return err
	}
	switch claim {
	case domain.ClaimAlreadySent:
		return nil
	case domain.ClaimInProgress:
		return domain.ErrNotificationInProgress
	}
	hits, err := uc.hitsFor(ctx, s, g.ID)
	if err != nil || len(hits) == 0 {
		return errors.Join(err, uc.log.Release(ctx, s.ID, g.ID))
	}
	err = uc.notifier.SendMatches(ctx, s, g, hits, domain.AlertIdempotencyKey(s.ID, g.ID))
	if err != nil && !errors.Is(err, domain.ErrAlreadyNotified) {
		return errors.Join(fmt.Errorf("notificar: %w", err), uc.log.Release(ctx, s.ID, g.ID))
	}
	return uc.log.MarkSent(ctx, s.ID, g.ID)
}

func (uc *MatchSubscriptions) hitsFor(ctx context.Context, s domain.Subscription, gazetteID string) ([]domain.ActHit, error) {
	if s.Entity != nil {
		return uc.acts.EntityHitsInGazette(ctx, gazetteID, *s.Entity)
	}
	if !s.Filter.IsZero() {
		return uc.filteredHits(ctx, s, gazetteID)
	}
	return uc.acts.SearchInGazette(ctx, gazetteID, domain.TranslateOperators(s.Query))
}

func (uc *MatchSubscriptions) filteredHits(ctx context.Context, s domain.Subscription, gazetteID string) ([]domain.ActHit, error) {
	f := s.Filter.ActFilter(s.Query, gazetteID)
	if err := f.Normalize(); err != nil {
		return nil, fmt.Errorf("filtro do alerta %s: %w", s.ID, err)
	}
	hits, _, err := uc.acts.Search(ctx, f)
	return hits, err
}
