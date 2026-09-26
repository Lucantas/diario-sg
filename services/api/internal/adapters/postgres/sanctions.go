package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const sanctionLinkRole = "sancionado"

type SanctionRepo struct{ db *sql.DB }

func NewSanctionRepo(db *sql.DB) *SanctionRepo { return &SanctionRepo{db: db} }

func (r *SanctionRepo) CitedCNPJs(ctx context.Context) ([]string, error) {
	return citedCNPJs(ctx, r.db)
}

func (r *SanctionRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM cgu_sanctions LIMIT 0`)
	return err
}

func (r *SanctionRepo) Save(ctx context.Context, load domain.SanctionLoad) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO cgu_sanctions (register, code, cnpj, cnpj_base, name, category, starts_at, ends_at, published_at,
			process, organ, organ_uf, sphere, scope, legal_basis, fine_cents, first_seen, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (register, code) DO UPDATE SET
			cnpj = excluded.cnpj, cnpj_base = excluded.cnpj_base, name = excluded.name, category = excluded.category,
			starts_at = excluded.starts_at, ends_at = excluded.ends_at, published_at = excluded.published_at,
			process = excluded.process, organ = excluded.organ, organ_uf = excluded.organ_uf, sphere = excluded.sphere,
			scope = excluded.scope, legal_basis = excluded.legal_basis, fine_cents = excluded.fine_cents,
			first_seen = least(cgu_sanctions.first_seen, excluded.first_seen),
			last_seen = greatest(cgu_sanctions.last_seen, excluded.last_seen)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, s := range load.Sanctions {
		if _, err := stmt.ExecContext(ctx, s.Register, s.Code, s.CNPJ, domain.CNPJBase(s.CNPJ), s.Name, s.Category,
			s.StartsAt, s.EndsAt, s.PublishedAt, s.Process, s.Organ, s.OrganUF, s.Sphere, s.Scope, s.LegalBasis,
			s.FineCents, s.FirstSeen, s.LastSeen); err != nil {
			return fmt.Errorf("sanção %s %s: %w", s.Register, s.Code, err)
		}
	}
	if err := relinkSanctions(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func relinkSanctions(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM entity_links WHERE source = $1`, domain.SourceSanctions); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO entity_links (entity_id, source, record_kind, record_id, role, certainty, evidence)
		SELECT e.id, $1, $2, s.register || ':' || s.code, $3,
		       CASE WHEN e.key = s.cnpj THEN $4 ELSE $5 END, s.category || ' — ' || s.organ
		FROM cgu_sanctions s
		JOIN entities e ON e.kind = 'cnpj' AND length(e.key) = 14 AND left(e.key, 8) = s.cnpj_base
		ON CONFLICT DO NOTHING`,
		domain.SourceSanctions, domain.RecordSanction, sanctionLinkRole, domain.CertaintyExact, domain.CertaintyStrong)
	return err
}

func (r *SanctionRepo) SanctionsListedOn(ctx context.Context) (map[string]time.Time, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT register, max(last_seen) FROM cgu_sanctions GROUP BY register`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]time.Time{}
	for rows.Next() {
		var register string
		var day time.Time
		if err := rows.Scan(&register, &day); err != nil {
			return nil, err
		}
		out[register] = day
	}
	return out, rows.Err()
}

func (r *SanctionRepo) SanctionsByCNPJ(ctx context.Context, cnpj string) ([]domain.Sanction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT register, code, cnpj, name, category, starts_at, ends_at, published_at, process, organ, organ_uf,
		       sphere, scope, legal_basis, fine_cents, first_seen, last_seen
		FROM cgu_sanctions WHERE cnpj_base = $1
		ORDER BY starts_at DESC NULLS LAST, register, code`, domain.CNPJBase(cnpj))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSanctions(rows)
}

func scanSanctions(rows *sql.Rows) ([]domain.Sanction, error) {
	out := []domain.Sanction{}
	for rows.Next() {
		var s domain.Sanction
		if err := rows.Scan(&s.Register, &s.Code, &s.CNPJ, &s.Name, &s.Category, &s.StartsAt, &s.EndsAt, &s.PublishedAt,
			&s.Process, &s.Organ, &s.OrganUF, &s.Sphere, &s.Scope, &s.LegalBasis, &s.FineCents, &s.FirstSeen, &s.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
