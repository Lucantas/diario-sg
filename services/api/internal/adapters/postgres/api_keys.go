package postgres

import (
	"context"
	"database/sql"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type APIKeyRepo struct{ db *sql.DB }

func NewAPIKeyRepo(db *sql.DB) *APIKeyRepo { return &APIKeyRepo{db: db} }

func (r *APIKeyRepo) Create(ctx context.Context, hash, prefix string) (domain.APIKey, error) {
	k := domain.APIKey{Prefix: prefix}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO api_keys (key_hash, key_prefix) VALUES ($1, $2) RETURNING id, created_at`, hash, prefix,
	).Scan(&k.ID, &k.CreatedAt)
	return k, err
}

func (r *APIKeyRepo) FindActive(ctx context.Context, hash string) (domain.APIKey, error) {
	var k domain.APIKey
	err := r.db.QueryRowContext(ctx, `
		SELECT id, key_prefix, created_at FROM api_keys WHERE key_hash = $1 AND revoked_at IS NULL`, hash,
	).Scan(&k.ID, &k.Prefix, &k.CreatedAt)
	return k, notFound(err)
}

func (r *APIKeyRepo) Revoke(ctx context.Context, hash string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE api_keys SET revoked_at = now() WHERE key_hash = $1 AND revoked_at IS NULL`, hash)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *APIKeyRepo) RecordUse(ctx context.Context, keyID, tool string) error {
	_, err := r.db.ExecContext(ctx, `
		WITH touched AS (UPDATE api_keys SET last_used_at = now() WHERE id = $1)
		INSERT INTO api_key_usage (key_id, day, tool, calls)
		VALUES ($1, (now() AT TIME ZONE 'America/Sao_Paulo')::date, $2, 1)
		ON CONFLICT (key_id, day, tool) DO UPDATE SET calls = api_key_usage.calls + 1`, keyID, tool)
	return err
}

func (r *APIKeyRepo) List(ctx context.Context) ([]domain.APIKey, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT k.id, k.key_prefix, k.created_at, coalesce(k.revoked_at, 'epoch'), coalesce(k.last_used_at, 'epoch'),
		       coalesce((SELECT sum(u.calls) FROM api_key_usage u
		                 WHERE u.key_id = k.id AND u.day > (now() AT TIME ZONE 'America/Sao_Paulo')::date - 30), 0)
		FROM api_keys k
		ORDER BY k.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.APIKey
	for rows.Next() {
		var k domain.APIKey
		if err := rows.Scan(&k.ID, &k.Prefix, &k.CreatedAt, &k.RevokedAt, &k.LastUsedAt, &k.RecentCalls); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *APIKeyRepo) RevokeByPrefix(ctx context.Context, prefix string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE api_keys SET revoked_at = now() WHERE key_prefix = $1 AND revoked_at IS NULL`, prefix)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
