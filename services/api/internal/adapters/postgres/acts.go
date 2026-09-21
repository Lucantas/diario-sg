package postgres

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type ActRepo struct{ db *sql.DB }

func NewActRepo(db *sql.DB) *ActRepo { return &ActRepo{db: db} }

func (r *ActRepo) Search(ctx context.Context, f domain.ActFilter) ([]domain.ActHit, int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, g.edition_number, g.published_at, g.source_url,
		       CASE WHEN $1 = '' THEN left(a.body, 280)
		            ELSE ts_headline('`+tsConfig+`', a.body, q, '`+headlineOpts+`') END,
		       `+cnpjsSubquery+`,
		       count(*) OVER ()
		FROM acts a
		JOIN gazettes g ON g.id = a.gazette_id
		CROSS JOIN websearch_to_tsquery('`+tsConfig+`', $1) q
		WHERE ($1 = '' OR `+matchFor("$1", "$7")+`)
		  AND ($2 = '' OR a.type = $2)
		  AND ($3::date IS NULL OR g.published_at >= $3::date)
		  AND ($4::date IS NULL OR g.published_at <= $4::date)
		ORDER BY CASE WHEN $1 = '' THEN 0 ELSE ts_rank(a.search, q) END DESC,
		         g.published_at DESC, a.position
		LIMIT $5 OFFSET $6`,
		f.Query, string(f.Type), nullableDate(f.From), nullableDate(f.To), f.Limit, f.Offset, likePattern(f.Query))
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		hits  []domain.ActHit
		total int
	)
	for rows.Next() {
		var h domain.ActHit
		var typ string
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Position, &h.EditionNumber, &h.PublishedAt, &h.SourceURL, &h.Snippet, pq.Array(&h.CNPJs), &total); err != nil {
			return nil, 0, err
		}
		h.Type = domain.ActType(typ)
		h.Snippet = highlightFallback(h.Snippet, f.Query)
		hits = append(hits, h)
	}
	return hits, total, rows.Err()
}

func (r *ActRepo) SearchInGazette(ctx context.Context, gazetteID, query string) ([]domain.ActHit, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, g.edition_number, g.published_at, g.source_url,
		       ts_headline('`+tsConfig+`', a.body, q, '`+headlineOpts+`'), `+cnpjsSubquery+`
		FROM acts a
		JOIN gazettes g ON g.id = a.gazette_id
		CROSS JOIN websearch_to_tsquery('`+tsConfig+`', $2) q
		WHERE a.gazette_id = $1 AND $2 <> '' AND `+matchFor("$2", "$3")+`
		ORDER BY ts_rank(a.search, q) DESC
		LIMIT 20`, gazetteID, query, likePattern(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []domain.ActHit
	for rows.Next() {
		var h domain.ActHit
		var typ string
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Position, &h.EditionNumber, &h.PublishedAt, &h.SourceURL, &h.Snippet, pq.Array(&h.CNPJs)); err != nil {
			return nil, err
		}
		h.Type = domain.ActType(typ)
		h.Snippet = highlightFallback(h.Snippet, query)
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

func (r *ActRepo) ListByGazette(ctx context.Context, gazetteID string) ([]domain.Act, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, gazette_id, type, title, body, position
		FROM acts WHERE gazette_id = $1 ORDER BY position`, gazetteID)
	if err != nil {
		return nil, notFound(err)
	}
	defer rows.Close()

	var acts []domain.Act
	for rows.Next() {
		var a domain.Act
		var typ string
		if err := rows.Scan(&a.ID, &a.GazetteID, &typ, &a.Title, &a.Body, &a.Position); err != nil {
			return nil, err
		}
		a.Type = domain.ActType(typ)
		acts = append(acts, a)
	}
	return acts, rows.Err()
}
