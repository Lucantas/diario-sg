//go:build integration

package integration

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/email"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

type slowSender struct {
	alerts atomic.Int32
	keys   sync.Map
}

func (s *slowSender) Send(_ context.Context, m email.Message) error {
	time.Sleep(50 * time.Millisecond)
	s.alerts.Add(1)
	s.keys.Store(m.IdempotencyKey, true)
	return nil
}

func confirmedAlert(t *testing.T, db *sql.DB, query string) domain.Subscription {
	t.Helper()
	ctx := context.Background()
	subscriptions := usecase.NewSubscriptions(postgres.NewSubscriptionRepo(db), email.NewNotifier(&inbox{}, "https://web.exemplo"))
	s, err := subscriptions.Subscribe(ctx, "rep@jornal.com", query, domain.AlertFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := subscriptions.Confirm(ctx, s.ConfirmToken); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestConcurrentDeliveriesOfTheSameEventSendOneEmail(t *testing.T) {
	_, db := newServerFor(t, filteredAlertGazette)
	ctx := context.Background()
	confirmedAlert(t, db, "loteamento")
	gazetteID := onlyGazetteID(t, db)

	for round := 0; round < 10; round++ {
		if _, err := db.ExecContext(ctx, `DELETE FROM notifications_sent`); err != nil {
			t.Fatal(err)
		}
		sender := &slowSender{}
		match := usecase.NewMatchSubscriptions(postgres.NewGazetteRepo(db), postgres.NewActRepo(db),
			postgres.NewSubscriptionRepo(db), postgres.NewNotificationLog(db), email.NewNotifier(sender, "https://web.exemplo"))
		start := make(chan struct{})
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for i := range errs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				errs[i] = match.Execute(ctx, gazetteID)
			}()
		}

		close(start)
		wg.Wait()

		if n := sender.alerts.Load(); n != 1 {
			t.Fatalf("rodada %d: esperava 1 e-mail, vieram %d (erros %v)", round, n, errs)
		}
		inProgress := 0
		for _, err := range errs {
			switch {
			case errors.Is(err, domain.ErrNotificationInProgress):
				inProgress++
			case err != nil:
				t.Fatalf("rodada %d: erro inesperado %v", round, err)
			}
		}
		if inProgress != 1 {
			t.Fatalf("rodada %d: a entrega que perdeu a reserva deve pedir nova tentativa: %v", round, errs)
		}
		if err := match.Execute(ctx, gazetteID); err != nil || sender.alerts.Load() != 1 {
			t.Fatalf("rodada %d: reentrega depois do envio não pode reenviar: %v, %d e-mails", round, err, sender.alerts.Load())
		}
	}
}

func TestNotificationLogClaimLifecycle(t *testing.T) {
	_, db := newServerFor(t, filteredAlertGazette)
	ctx := context.Background()
	sub := confirmedAlert(t, db, "loteamento")
	gazetteID := onlyGazetteID(t, db)
	log := postgres.NewNotificationLog(db)
	claim := func(want domain.NotificationClaim, step string) {
		t.Helper()
		got, err := log.Claim(ctx, sub.ID, gazetteID)
		if err != nil || got != want {
			t.Fatalf("%s: reserva %v, esperava %v (erro %v)", step, got, want, err)
		}
	}

	claim(domain.ClaimAcquired, "primeira reserva")
	claim(domain.ClaimInProgress, "reserva em andamento")
	if err := log.Release(ctx, sub.ID, gazetteID); err != nil {
		t.Fatal(err)
	}
	claim(domain.ClaimAcquired, "depois de liberar")
	if _, err := db.ExecContext(ctx, `UPDATE notifications_sent SET claimed_at = now() - interval '1 hour'`); err != nil {
		t.Fatal(err)
	}
	claim(domain.ClaimAcquired, "reserva vencida")
	claim(domain.ClaimInProgress, "reserva renovada")
	if err := log.MarkSent(ctx, sub.ID, gazetteID); err != nil {
		t.Fatal(err)
	}
	claim(domain.ClaimAlreadySent, "depois de enviado")
	if err := log.Release(ctx, sub.ID, gazetteID); err != nil {
		t.Fatal(err)
	}
	claim(domain.ClaimAlreadySent, "liberar não desfaz um envio")
}
