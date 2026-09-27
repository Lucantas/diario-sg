package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type OAuthRepo struct{ db *sql.DB }

func NewOAuthRepo(db *sql.DB) *OAuthRepo { return &OAuthRepo{db: db} }

func (r *OAuthRepo) CreateClient(ctx context.Context, c domain.OAuthClient, unusedBefore time.Time) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM oauth_clients WHERE last_used_at IS NULL AND created_at < $1`, unusedBefore); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO oauth_clients (id, name, redirect_uris, created_at) VALUES ($1, $2, $3, $4)`,
		c.ID, c.Name, pq.Array(c.RedirectURIs), c.CreatedAt)
	return err
}

func (r *OAuthRepo) Client(ctx context.Context, id string) (domain.OAuthClient, error) {
	var c domain.OAuthClient
	err := r.db.QueryRowContext(ctx, `SELECT id, name, redirect_uris, created_at FROM oauth_clients WHERE id = $1`, id).
		Scan(&c.ID, &c.Name, pq.Array(&c.RedirectURIs), &c.CreatedAt)
	return c, notFound(err)
}

func (r *OAuthRepo) MarkClientUsed(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE oauth_clients SET last_used_at = $2 WHERE id = $1`, id, at)
	return err
}

func (r *OAuthRepo) SaveCode(ctx context.Context, code domain.OAuthCode, now time.Time) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM oauth_codes WHERE expires_at < $1`, now); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oauth_codes (code_hash, client_id, redirect_uri, challenge, resource, expires_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		code.Hash, code.ClientID, code.RedirectURI, code.Challenge, code.Resource, code.ExpiresAt)
	return err
}

func (r *OAuthRepo) TakeCode(ctx context.Context, hash string) (domain.OAuthCode, error) {
	var c domain.OAuthCode
	err := r.db.QueryRowContext(ctx, `
		DELETE FROM oauth_codes WHERE code_hash = $1 RETURNING code_hash, client_id, redirect_uri, challenge, resource, expires_at`, hash).
		Scan(&c.Hash, &c.ClientID, &c.RedirectURI, &c.Challenge, &c.Resource, &c.ExpiresAt)
	return c, notFound(err)
}
