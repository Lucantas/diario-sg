package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type SubscriptionRepo struct{ db *sql.DB }

func NewSubscriptionRepo(db *sql.DB) *SubscriptionRepo { return &SubscriptionRepo{db: db} }

const subCols = `id, email, coalesce(query, ''), entity_kind, entity_key, entity_label, status, confirm_token, unsubscribe_token, created_at, confirmed_at,
	filter_source, filter_type, filter_organ, filter_theme`

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
		INSERT INTO subscriptions (email, query, entity_kind, entity_key, entity_label, status, confirm_token, unsubscribe_token, created_at,
			filter_source, filter_type, filter_organ, filter_theme)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id`,
		s.Email, query, kind, key, label, string(s.Status), s.ConfirmToken, s.UnsubscribeToken, s.CreatedAt,
		s.Filter.Source, string(s.Filter.Type), s.Filter.Organ, s.Filter.Theme,
	).Scan(&s.ID)
}

func (r *SubscriptionRepo) FindByConfirmToken(ctx context.Context, token string) (domain.Subscription, error) {
	return r.findOne(ctx, `SELECT `+subCols+` FROM subscriptions WHERE confirm_token = $1`, token)
}

func (r *SubscriptionRepo) FindByUnsubscribeToken(ctx context.Context, token string) (domain.Subscription, error) {
	return r.findOne(ctx, `SELECT `+subCols+` FROM subscriptions WHERE unsubscribe_token = $1`, token)
}

func (r *SubscriptionRepo) Update(ctx context.Context, s domain.Subscription) error {
	_, err := r.db.ExecContext(ctx, `UPDATE subscriptions SET status = $2, confirmed_at = $3 WHERE id = $1`,
		s.ID, string(s.Status), s.ConfirmedAt)
	return err
}

func (r *SubscriptionRepo) ListActive(ctx context.Context) ([]domain.Subscription, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+subCols+` FROM subscriptions WHERE status = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Subscription
	for rows.Next() {
		s, err := scanSub(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *SubscriptionRepo) findOne(ctx context.Context, q, arg string) (domain.Subscription, error) {
	s, err := scanSub(r.db.QueryRowContext(ctx, q, arg))
	return s, notFound(err)
}

type scanner interface{ Scan(dest ...any) error }

func scanSub(row scanner) (domain.Subscription, error) {
	var s domain.Subscription
	var status, filterType string
	var kind, key, label sql.NullString
	var confirmed sql.NullTime
	err := row.Scan(&s.ID, &s.Email, &s.Query, &kind, &key, &label, &status, &s.ConfirmToken, &s.UnsubscribeToken, &s.CreatedAt, &confirmed,
		&s.Filter.Source, &filterType, &s.Filter.Organ, &s.Filter.Theme)
	s.Filter.Type = domain.ActType(filterType)
	s.Status = domain.SubscriptionStatus(status)
	if kind.Valid {
		s.Entity = &domain.EntityRef{Kind: domain.EntityKind(kind.String), Key: key.String, Label: label.String}
	}
	if confirmed.Valid {
		s.ConfirmedAt = &confirmed.Time
	}
	return s, err
}

type NotificationLog struct{ db *sql.DB }

func NewNotificationLog(db *sql.DB) *NotificationLog { return &NotificationLog{db: db} }

const notificationClaimLease = 2 * time.Minute

func (n *NotificationLog) Claim(ctx context.Context, subID, gazetteID string) (domain.NotificationClaim, error) {
	var claimed bool
	var status sql.NullString
	err := n.db.QueryRowContext(ctx, `
		WITH claimed AS (
			INSERT INTO notifications_sent (subscription_id, gazette_id, status, claimed_at, sent_at)
			VALUES ($1, $2, 'pending', now(), NULL)
			ON CONFLICT (subscription_id, gazette_id) DO UPDATE SET claimed_at = now()
			WHERE notifications_sent.status = 'pending'
			  AND notifications_sent.claimed_at < now() - $3 * interval '1 second'
			RETURNING 1
		)
		SELECT EXISTS (SELECT 1 FROM claimed),
		       (SELECT status FROM notifications_sent WHERE subscription_id = $1 AND gazette_id = $2)`,
		subID, gazetteID, notificationClaimLease.Seconds()).Scan(&claimed, &status)
	switch {
	case err != nil:
		return 0, err
	case claimed:
		return domain.ClaimAcquired, nil
	case status.String == "sent":
		return domain.ClaimAlreadySent, nil
	default:
		return domain.ClaimInProgress, nil
	}
}

func (n *NotificationLog) MarkSent(ctx context.Context, subID, gazetteID string) error {
	_, err := n.db.ExecContext(ctx, `
		INSERT INTO notifications_sent (subscription_id, gazette_id, status, claimed_at, sent_at)
		VALUES ($1, $2, 'sent', now(), now())
		ON CONFLICT (subscription_id, gazette_id) DO UPDATE SET status = 'sent', sent_at = now()`, subID, gazetteID)
	return err
}

func (n *NotificationLog) Release(ctx context.Context, subID, gazetteID string) error {
	_, err := n.db.ExecContext(ctx, `DELETE FROM notifications_sent
		WHERE subscription_id = $1 AND gazette_id = $2 AND status = 'pending'`, subID, gazetteID)
	return err
}
