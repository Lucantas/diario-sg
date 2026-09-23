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
	where, extra := filterSQL(f, 5)
	args := append([]any{f.Query, f.Limit, f.Offset, likePattern(f.Query)}, extra...)
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, a.organ, coalesce(a.page_start, 0), coalesce(a.page_end, 0),
		       g.edition_number, g.published_at, g.is_extra, g.source_url, g.checksum,
		       CASE WHEN $1 = '' THEN left(a.body, 280)
		            ELSE ts_headline('`+tsConfig+`', a.body, q, '`+headlineOpts+`') END,
		       `+cnpjsSubquery+`,
		       `+valuesSubquery+`,
		       count(*) OVER ()
		FROM acts a
		JOIN gazettes g ON g.id = a.gazette_id
		CROSS JOIN websearch_to_tsquery('`+tsConfig+`', $1) q
		`+exactPhraseFor("$4")+`
		WHERE ($1 = '' OR `+matchFor("$1", "$4")+`)`+where+`
		`+orderSQL(f)+`
		LIMIT $2 OFFSET $3`, args...)
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
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Position, &h.Organ, &h.PageStart, &h.PageEnd,
			&h.EditionNumber, &h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Checksum, &h.Snippet, pq.Array(&h.CNPJs), pq.Array(&h.ValuesCents), &total); err != nil {
			return nil, 0, err
		}
		h.Type = domain.ActType(typ)
		h.Snippet = highlightFallback(h.Snippet, phraseOf(f.Query))
		hits = append(hits, h)
	}
	return hits, total, rows.Err()
}

func (r *ActRepo) SearchInGazette(ctx context.Context, gazetteID, query string) ([]domain.ActHit, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, g.edition_number, g.published_at, g.is_extra, g.source_url,
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
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Position, &h.EditionNumber, &h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Snippet, pq.Array(&h.CNPJs)); err != nil {
			return nil, err
		}
		h.Type = domain.ActType(typ)
		h.Snippet = highlightFallback(h.Snippet, phraseOf(query))
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

func (r *ActRepo) ListByGazette(ctx context.Context, gazetteID string) ([]domain.Act, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, gazette_id, type, title, body, position, coalesce(page_start, 0), coalesce(page_end, 0), organ
		FROM acts WHERE gazette_id = $1 ORDER BY position`, gazetteID)
	if err != nil {
		return nil, notFound(err)
	}
	defer rows.Close()

	var acts []domain.Act
	for rows.Next() {
		var a domain.Act
		var typ string
		if err := rows.Scan(&a.ID, &a.GazetteID, &typ, &a.Title, &a.Body, &a.Position, &a.PageStart, &a.PageEnd, &a.Organ); err != nil {
			return nil, err
		}
		a.Type = domain.ActType(typ)
		acts = append(acts, a)
	}
	return acts, rows.Err()
}

func (r *ActRepo) Export(ctx context.Context, f domain.ActFilter, yield func(domain.ActHit, int) error) error {
	where, extra := filterSQL(f, 4)
	args := append([]any{f.Query, f.Limit, likePattern(f.Query)}, extra...)
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.body, a.position, a.organ, coalesce(a.page_start, 0), coalesce(a.page_end, 0),
		       g.edition_number, g.published_at, g.is_extra, g.source_url, g.checksum,
		       `+cnpjsSubquery+`,
		       `+valuesSubquery+`,
		       count(*) OVER ()
		FROM acts a
		JOIN gazettes g ON g.id = a.gazette_id
		CROSS JOIN websearch_to_tsquery('`+tsConfig+`', $1) q
		`+exactPhraseFor("$3")+`
		WHERE ($1 = '' OR `+matchFor("$1", "$3")+`)`+where+`
		`+orderSQL(f)+`
		LIMIT $2`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var h domain.ActHit
		var typ string
		var total int
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Body, &h.Position, &h.Organ, &h.PageStart, &h.PageEnd,
			&h.EditionNumber, &h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Checksum,
			pq.Array(&h.CNPJs), pq.Array(&h.ValuesCents), &total); err != nil {
			return err
		}
		h.Type = domain.ActType(typ)
		if err := yield(h, total); err != nil {
			return err
		}
	}
	return rows.Err()
}
