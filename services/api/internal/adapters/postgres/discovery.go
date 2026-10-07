package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type DiscoveryRepo struct{ db *sql.DB }

func NewDiscoveryRepo(db *sql.DB) *DiscoveryRepo { return &DiscoveryRepo{db: db} }

const companyColumns = `e.key, coalesce(nullif(max(co.name), ''), max(es.trade_name), ''), count(DISTINCT l.record_id)`

const companyJoins = `
	FROM entities e
	JOIN entity_links l ON l.entity_id = e.id AND l.record_kind = '` + domain.RecordAct + `'
	LEFT JOIN rf_establishments es ON es.cnpj = e.key
	LEFT JOIN rf_companies co ON co.cnpj_base = es.cnpj_base`

func (r *DiscoveryRepo) CompanyByCNPJ(ctx context.Context, cnpj string) (domain.Suggestion, bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+companyColumns+companyJoins+`
		WHERE e.kind = $1 AND e.key = $2
		GROUP BY e.key`, string(domain.EntityCNPJ), cnpj)
	if err != nil {
		return domain.Suggestion{}, false, err
	}
	companies, err := scanCompanies(rows)
	if err != nil || len(companies) == 0 {
		return domain.Suggestion{}, false, err
	}
	return companies[0], true, nil
}

func (r *DiscoveryRepo) CompaniesByName(ctx context.Context, text string, limit int) ([]domain.Suggestion, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+companyColumns+companyJoins+`
		WHERE e.kind = $1
		  AND (unaccent_immutable(co.name) ILIKE unaccent_immutable($2) OR unaccent_immutable(es.trade_name) ILIKE unaccent_immutable($2))
		GROUP BY e.key
		ORDER BY count(DISTINCT l.record_id) DESC, 2
		LIMIT $3`, string(domain.EntityCNPJ), domain.ContainsPattern(text), limit)
	if err != nil {
		return nil, err
	}
	return scanCompanies(rows)
}

func scanCompanies(rows *sql.Rows) ([]domain.Suggestion, error) {
	defer rows.Close()
	var out []domain.Suggestion
	for rows.Next() {
		s := domain.Suggestion{Kind: domain.EntityCNPJ}
		if err := rows.Scan(&s.Key, &s.Name, &s.Acts); err != nil {
			return nil, err
		}
		s.Label = domain.FormatCNPJ(s.Key)
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *DiscoveryRepo) EntityActs(ctx context.Context, kind domain.EntityKind, key string) (int, error) {
	var acts int
	err := r.db.QueryRowContext(ctx, `
		SELECT count(DISTINCT l.record_id)
		FROM entities e JOIN entity_links l ON l.entity_id = e.id AND l.record_kind = $3
		WHERE e.kind = $1 AND e.key = $2`, string(kind), key, domain.RecordAct).Scan(&acts)
	return acts, err
}

func (r *DiscoveryRepo) LatestEditions(ctx context.Context) ([]domain.LatestEdition, error) {
	editions, err := r.lastDayEditions(ctx)
	if err != nil || len(editions) == 0 {
		return editions, err
	}
	ids := make([]string, len(editions))
	byID := make(map[string]*domain.LatestEdition, len(editions))
	for i := range editions {
		ids[i] = editions[i].GazetteID
		byID[editions[i].GazetteID] = &editions[i]
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT gazette_id, type, count(*), coalesce(sum(main_value_cents), 0)
		FROM acts WHERE gazette_id = ANY($1::uuid[])
		GROUP BY gazette_id, type`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var t domain.TypeTotal
		if err := rows.Scan(&id, &t.Type, &t.Acts, &t.ValueCents); err != nil {
			return nil, err
		}
		byID[id].Types = append(byID[id].Types, t)
	}
	return editions, rows.Err()
}

func (r *DiscoveryRepo) lastDayEditions(ctx context.Context) ([]domain.LatestEdition, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH last AS (SELECT source, max(published_at) AS day FROM gazettes GROUP BY source)
		SELECT g.id, g.source, g.published_at, g.edition_number, g.is_extra
		FROM gazettes g
		JOIN last ON last.source = g.source AND last.day = g.published_at
		ORDER BY g.source_url`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LatestEdition
	for rows.Next() {
		var e domain.LatestEdition
		var day time.Time
		if err := rows.Scan(&e.GazetteID, &e.Source, &day, &e.EditionNumber, &e.IsExtra); err != nil {
			return nil, err
		}
		e.Day = day
		out = append(out, e)
	}
	return out, rows.Err()
}
