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
		sent, err := uc.log.WasSent(ctx, s.ID, g.ID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if sent {
			continue
		}
		hits, err := uc.acts.SearchInGazette(ctx, g.ID, domain.TranslateOperators(s.Query))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(hits) == 0 {
			continue
		}
		if err := uc.notifier.SendMatches(ctx, s, g, hits); err != nil {
			errs = append(errs, fmt.Errorf("notificar %s: %w", s.ID, err))
			continue
		}
		if err := uc.log.MarkSent(ctx, s.ID, g.ID); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
