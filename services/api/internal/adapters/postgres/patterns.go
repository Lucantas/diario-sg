package postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type PatternRepo struct{ db *sql.DB }

func NewPatternRepo(db *sql.DB) *PatternRepo { return &PatternRepo{db: db} }

func (r *PatternRepo) DispensaActs(ctx context.Context) ([]domain.DispensaAct, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, e.key, a.organ, g.published_at, a.main_value_cents, a.body,
		       coalesce((SELECT array_agg(p.key || '|' || pl.evidence ORDER BY p.key)
		                 FROM entity_links pl JOIN entities p ON p.id = pl.entity_id AND p.kind = 'processo'
		                 WHERE pl.source = $2 AND pl.record_kind = $1 AND pl.record_id = a.id::text), '{}'),
		       coalesce((SELECT array_agg(o.key ORDER BY o.key)
		                 FROM entity_links ol JOIN entities o ON o.id = ol.entity_id AND o.kind = 'cnpj'
		                 WHERE ol.source = $2 AND ol.record_kind = $1 AND ol.record_id = a.id::text AND o.key <> e.key), '{}')
		FROM acts a
		JOIN gazettes g ON g.id = a.gazette_id
		JOIN entity_links l ON l.source = $2 AND l.record_kind = $1 AND l.record_id = a.id::text
		JOIN entities e ON e.id = l.entity_id AND e.kind = 'cnpj'
		WHERE (a.type = 'dispensa' OR a.modality = 'dispensa') AND a.type <> 'aditivo'
		  AND a.main_value_cents > 0 AND g.source = $2`, domain.RecordAct, domain.SourceDiarioPrefeitura)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.DispensaAct
	for rows.Next() {
		var a domain.DispensaAct
		var processes []string
		if err := rows.Scan(&a.ActID, &a.CNPJ, &a.Organ, &a.PublishedAt, &a.ValueCents, &a.Body, pq.Array(&processes), pq.Array(&a.OtherCNPJs)); err != nil {
			return nil, err
		}
		for _, p := range processes {
			key, evidence, _ := strings.Cut(p, "|")
			a.Processes = append(a.Processes, domain.DispensaProcess{Key: key, Label: domain.EntityLabel(domain.EntityProcesso, evidence)})
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PatternRepo) MonthlyActCounts(ctx context.Context, types []domain.ActType, source string) ([]domain.MonthlyActCount, error) {
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = string(t)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.type, extract(year FROM g.published_at)::int, extract(month FROM g.published_at)::int, count(*)
		FROM acts a JOIN gazettes g ON g.id = a.gazette_id
		WHERE a.type = ANY($1) AND g.source = $2 AND g.published_at < date_trunc('month', now())
		GROUP BY 1, 2, 3`, pq.Array(names), source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MonthlyActCount
	for rows.Next() {
		var c domain.MonthlyActCount
		var typ string
		var month int
		if err := rows.Scan(&typ, &c.Year, &month, &c.Count); err != nil {
			return nil, err
		}
		c.Type, c.Month = domain.ActType(typ), time.Month(month)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PatternRepo) HitsByIDs(ctx context.Context, ids []string) ([]domain.ActHit, error) {
	return hitsByIDs(ctx, r.db, ids)
}

func hitsByIDs(ctx context.Context, db *sql.DB, ids []string) ([]domain.ActHit, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, a.organ, coalesce(a.page_start, 0), coalesce(a.page_end, 0),
		       coalesce(a.modality, ''), coalesce(a.main_value_cents, 0),
		       g.edition_number, g.published_at, g.is_extra, g.source_url, g.checksum, g.source,
		       left(a.body, 280), `+cnpjsSubquery+`, `+valuesSubquery+`, `+mentionsSubquery+`
		FROM acts a JOIN gazettes g ON g.id = a.gazette_id
		WHERE a.id = ANY($1::uuid[])
		ORDER BY g.published_at, a.position`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []domain.ActHit
	for rows.Next() {
		var h domain.ActHit
		var typ string
		var mentions []string
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Position, &h.Organ, &h.PageStart, &h.PageEnd,
			&h.Modality, &h.MainValueCents, &h.EditionNumber, &h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Checksum, &h.Source,
			&h.Snippet, pq.Array(&h.CNPJs), pq.Array(&h.ValuesCents), pq.Array(&mentions)); err != nil {
			return nil, err
		}
		h.Type = domain.ActType(typ)
		h.Mentions = parseMentions(mentions)
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hits, markBodyFacts(ctx, db, hits)
}
