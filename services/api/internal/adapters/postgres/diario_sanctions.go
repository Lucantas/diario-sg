package postgres

import (
	"context"
	"database/sql"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const sanctionCandidateFilter = `(a.title ~* '(advert[êe]ncia|multa|san[çc][ãa]o|penalidade)' OR
	a.body ~* '(aplic|imp[oô]|decid)[a-zçãõ-]* [^.;]{0,60}(penalidade|san[çc][ãa]o|pena|multa)[^.;]{0,40}de (multa|advert|suspens|impedimento|declara|inidon)')`

type DiarioSanctionRepo struct{ db *sql.DB }

func NewDiarioSanctionRepo(db *sql.DB) *DiarioSanctionRepo { return &DiarioSanctionRepo{db: db} }

func (r *DiarioSanctionRepo) SanctionCandidatesByCNPJ(ctx context.Context, cnpj string) ([]domain.SanctionCandidate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.title, a.body
		FROM entities e
		JOIN entity_links l ON l.entity_id = e.id AND l.record_kind = $2
		JOIN acts a ON a.id = l.record_id::uuid
		WHERE e.kind = 'cnpj' AND e.key = $1 AND `+sanctionCandidateFilter, cnpj, domain.RecordAct)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SanctionCandidate
	for rows.Next() {
		var c domain.SanctionCandidate
		if err := rows.Scan(&c.ActID, &c.Title, &c.Body); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *DiarioSanctionRepo) HitsByIDs(ctx context.Context, ids []string) ([]domain.ActHit, error) {
	return hitsByIDs(ctx, r.db, ids)
}
