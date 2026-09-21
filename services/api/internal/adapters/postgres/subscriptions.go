package postgres

import (
	"context"
	"database/sql"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type SubscriptionRepo struct{ db *sql.DB }

func NewSubscriptionRepo(db *sql.DB) *SubscriptionRepo { return &SubscriptionRepo{db: db} }

const subCols = `id, email, query, status, confirm_token, unsubscribe_token, created_at, confirmed_at`

func (r *SubscriptionRepo) Create(ctx context.Context, s *domain.Subscription) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO subscriptions (email, query, status, confirm_token, unsubscribe_token, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		s.Email, s.Query, string(s.Status), s.ConfirmToken, s.UnsubscribeToken, s.CreatedAt,
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
	var status string
	var confirmed sql.NullTime
	err := row.Scan(&s.ID, &s.Email, &s.Query, &status, &s.ConfirmToken, &s.UnsubscribeToken, &s.CreatedAt, &confirmed)
	s.Status = domain.SubscriptionStatus(status)
	if confirmed.Valid {
		s.ConfirmedAt = &confirmed.Time
	}
	return s, err
}

// NotificationLog implementa ports.NotificationLog.
type NotificationLog struct{ db *sql.DB }

func NewNotificationLog(db *sql.DB) *NotificationLog { return &NotificationLog{db: db} }

func (n *NotificationLog) WasSent(ctx context.Context, subID, gazetteID string) (bool, error) {
	var ok bool
	err := n.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM notifications_sent
		WHERE subscription_id = $1 AND gazette_id = $2)`, subID, gazetteID).Scan(&ok)
	return ok, err
}

func (n *NotificationLog) MarkSent(ctx context.Context, subID, gazetteID string) error {
	_, err := n.db.ExecContext(ctx, `INSERT INTO notifications_sent (subscription_id, gazette_id)
		VALUES ($1, $2) ON CONFLICT DO NOTHING`, subID, gazetteID)
	return err
}
