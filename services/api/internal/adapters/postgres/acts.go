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
		WITH matched AS MATERIALIZED (`+matchedSQL(where)+`), page AS (
			SELECT id, exact, CASE WHEN exact THEN 0 ELSE ts END AS rank, published_at, source_url, position, count(*) OVER () AS total
			FROM matched
			`+matchedOrderSQL(f)+`
			LIMIT $2 OFFSET $3
		)
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, a.organ, coalesce(a.page_start, 0), coalesce(a.page_end, 0),
		       g.edition_number, g.published_at, g.is_extra, g.source_url, g.checksum, g.source,
		       CASE WHEN $1 = '' THEN left(a.body, 280)
		            ELSE ts_headline('`+tsConfig+`', a.body, q, '`+headlineOpts+`') END,
		       `+cnpjsSubquery+`,
		       `+valuesSubquery+`,
		       p.total
		FROM page p
		JOIN acts a ON a.id = p.id
		JOIN gazettes g ON g.id = a.gazette_id
		CROSS JOIN websearch_to_tsquery('`+tsConfig+`', $1) q
		`+pageOrderSQL(f), args...)
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
			&h.EditionNumber, &h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Checksum, &h.Source, &h.Snippet, pq.Array(&h.CNPJs), pq.Array(&h.ValuesCents), &total); err != nil {
			return nil, 0, err
		}
		h.Type = domain.ActType(typ)
		h.Snippet = highlightFallback(h.Snippet, phraseOf(f.Query))
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(hits) == 0 && f.Offset > 0 {
		if total, err = r.countMatched(ctx, where, args); err != nil {
			return nil, 0, err
		}
	}
	return hits, total, markBodyFacts(ctx, r.db, hits)
}

func matchedSQL(where string) string {
	return `
			SELECT a.id, ($1 <> '' AND ` + exactPhraseExpr("$4") + `) AS exact,
			       CASE WHEN $1 = '' THEN 0 ELSE ts_rank(a.search, q) END AS ts,
			       g.published_at, g.source_url, a.position
			FROM acts a
			JOIN gazettes g ON g.id = a.gazette_id
			CROSS JOIN websearch_to_tsquery('` + tsConfig + `', $1) q
			WHERE ($1 = '' OR ` + matchFor("$1", "$4") + `)` + where
}

func (r *ActRepo) countMatched(ctx context.Context, where string, args []any) (int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `
		SELECT count(*) FROM (`+matchedSQL(where)+`) m
		WHERE $2::int IS NOT NULL AND $3::int IS NOT NULL`, args...).Scan(&total)
	return total, err
}

func (r *ActRepo) SearchInGazette(ctx context.Context, gazetteID, query string) ([]domain.ActHit, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, g.edition_number, g.published_at, g.is_extra, g.source_url, g.source,
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
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Position, &h.EditionNumber, &h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Source, &h.Snippet, pq.Array(&h.CNPJs)); err != nil {
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
		       g.edition_number, g.published_at, g.is_extra, g.source_url, g.checksum, g.source,
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
			&h.EditionNumber, &h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Checksum, &h.Source,
			pq.Array(&h.CNPJs), pq.Array(&h.ValuesCents), &total); err != nil {
			return err
		}
		h.Type = domain.ActType(typ)
		f := domain.WarningFactsOf(h.Act)
		h.TitleOnly, h.Signatures = f.TitleOnly, f.Signatures
		if err := yield(h, total); err != nil {
			return err
		}
	}
	return rows.Err()
}
