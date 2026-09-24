package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func linkActs(ctx context.Context, tx *sql.Tx, gazetteID string) error {
	_, err := tx.ExecContext(ctx, `SELECT link_diario_acts($1)`, gazetteID)
	return err
}

func unlinkActs(ctx context.Context, tx *sql.Tx, gazetteID string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM entity_links
		WHERE record_kind = $2
		  AND record_id IN (SELECT id::text FROM acts WHERE gazette_id = $1)`,
		gazetteID, domain.RecordAct)
	return err
}

type LinkRepo struct{ db *sql.DB }

func NewLinkRepo(db *sql.DB) *LinkRepo { return &LinkRepo{db: db} }

const weakestCertainty = `(ARRAY['fraca', 'forte', 'exata'])[min(array_position(ARRAY['fraca', 'forte', 'exata'], l.certainty))]`

func (r *LinkRepo) ReportByKey(ctx context.Context, kind domain.EntityKind, key, source string) (domain.EntityReport, error) {
	report := domain.EntityReport{Kind: kind, Key: key, CountByType: map[domain.ActType]int{}}
	var entityID, certainty string
	err := r.db.QueryRowContext(ctx, `
		SELECT e.id, `+weakestCertainty+`, count(DISTINCT l.source)
		FROM entities e JOIN entity_links l ON l.entity_id = e.id AND l.record_kind = $3 AND ($4 = '' OR l.source = $4)
		WHERE e.kind = $1 AND e.key = $2
		GROUP BY e.id`, string(kind), key, domain.RecordAct, source).Scan(&entityID, &certainty, &report.Sources)
	if errors.Is(err, sql.ErrNoRows) {
		return report, nil
	}
	if err != nil {
		return report, err
	}
	report.Certainty = domain.ReportCertainty(kind, domain.Certainty(certainty), report.Sources)
	if err := r.linkedActs(ctx, entityID, source, &report); err != nil {
		return report, err
	}
	if err := r.linkedTypes(ctx, entityID, source, &report); err != nil {
		return report, err
	}
	if kind != domain.EntityProcesso {
		if err := r.linkedProcesses(ctx, entityID, source, &report); err != nil {
			return report, err
		}
	}
	err = r.db.QueryRowContext(ctx, `
		SELECT coalesce(sum(v.normalized::bigint), 0)
		FROM entity_links l JOIN act_entities v ON v.act_id = l.record_id::uuid AND v.kind = 'valor'
		WHERE l.entity_id = $1 AND l.record_kind = $2 AND ($3 = '' OR l.source = $3)`,
		entityID, domain.RecordAct, source).Scan(&report.TotalCents)
	return report, err
}

func (r *LinkRepo) linkedActs(ctx context.Context, entityID, source string, report *domain.EntityReport) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, a.organ, coalesce(a.page_start, 0), coalesce(a.page_end, 0),
		       coalesce(a.modality, ''), coalesce(a.main_value_cents, 0),
		       g.edition_number, g.published_at, g.is_extra, g.source_url, g.checksum, g.source,
		       substr(a.body, greatest(position(l.evidence IN a.body) - $3, 1), 2 * $3 + length(l.evidence)), l.evidence,
		       `+mentionsSubquery+`
		FROM entity_links l
		JOIN acts a ON a.id = l.record_id::uuid
		JOIN gazettes g ON g.id = a.gazette_id
		WHERE l.entity_id = $1 AND l.record_kind = $2 AND ($5 = '' OR l.source = $5)
		ORDER BY g.published_at DESC, a.position
		LIMIT $4`, entityID, domain.RecordAct, snippetRadius, reportActsLimit, source)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var h domain.ActHit
		var typ, evidence string
		var mentions []string
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Position, &h.Organ, &h.PageStart, &h.PageEnd,
			&h.Modality, &h.MainValueCents, &h.EditionNumber, &h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Checksum, &h.Source, &h.Snippet, &evidence, pq.Array(&mentions)); err != nil {
			return err
		}
		h.Type = domain.ActType(typ)
		h.Snippet = highlightFallback(h.Snippet, evidence)
		h.Mentions = parseMentions(mentions)
		report.Acts = append(report.Acts, h)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return markBodyFacts(ctx, r.db, report.Acts)
}

func (r *LinkRepo) linkedTypes(ctx context.Context, entityID, source string, report *domain.EntityReport) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.type, count(*)
		FROM entity_links l JOIN acts a ON a.id = l.record_id::uuid
		WHERE l.entity_id = $1 AND l.record_kind = $2 AND ($3 = '' OR l.source = $3)
		GROUP BY a.type`, entityID, domain.RecordAct, source)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var typ string
		var n int
		if err := rows.Scan(&typ, &n); err != nil {
			return err
		}
		report.CountByType[domain.ActType(typ)] = n
		report.TotalActs += n
	}
	return rows.Err()
}

const processSummaryLimit = 20

func (r *LinkRepo) linkedProcesses(ctx context.Context, entityID, source string, report *domain.EntityReport) error {
	rows, err := r.db.QueryContext(ctx, `
		WITH acts_of AS (
			SELECT l.record_id, l.source FROM entity_links l
			WHERE l.entity_id = $1 AND l.record_kind = $2 AND ($3 = '' OR l.source = $3)
		), processes AS (
			SELECT pe.key, o.record_id
			FROM acts_of o
			JOIN entity_links lp ON lp.record_kind = $2 AND lp.record_id = o.record_id AND lp.source = o.source
			JOIN entities pe ON pe.id = lp.entity_id AND pe.kind = '`+string(domain.EntityProcesso)+`'
		)
		SELECT p.key, count(DISTINCT p.record_id),
		       coalesce(max((SELECT max(v.normalized::bigint) FROM act_entities v WHERE v.act_id = p.record_id::uuid AND v.kind = 'valor')), 0),
		       min(g.published_at), max(g.published_at),
		       (SELECT count(*) FROM acts_of o WHERE NOT EXISTS (SELECT 1 FROM processes q WHERE q.record_id = o.record_id))
		FROM processes p
		JOIN acts a ON a.id = p.record_id::uuid
		JOIN gazettes g ON g.id = a.gazette_id
		GROUP BY p.key
		ORDER BY count(DISTINCT p.record_id) DESC, p.key
		LIMIT $4`, entityID, domain.RecordAct, source, processSummaryLimit)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var p domain.ProcessSummary
		if err := rows.Scan(&p.Key, &p.Acts, &p.MaxValueCents, &p.First, &p.Last, &report.ActsWithoutProcess); err != nil {
			return err
		}
		report.ByProcess = append(report.ByProcess, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(report.ByProcess) == 0 {
		report.ActsWithoutProcess = report.TotalActs
		return nil
	}
	return r.processTotals(ctx, entityID, source, report)
}

func (r *LinkRepo) processTotals(ctx context.Context, entityID, source string, report *domain.EntityReport) error {
	return r.db.QueryRowContext(ctx, `
		WITH acts_of AS (
			SELECT l.record_id, l.source FROM entity_links l
			WHERE l.entity_id = $1 AND l.record_kind = $2 AND ($3 = '' OR l.source = $3)
		), processes AS (
			SELECT pe.key, o.record_id,
			       coalesce((SELECT max(v.normalized::bigint) FROM act_entities v WHERE v.act_id = o.record_id::uuid AND v.kind = 'valor'), 0) AS value
			FROM acts_of o
			JOIN entity_links lp ON lp.record_kind = $2 AND lp.record_id = o.record_id AND lp.source = o.source
			JOIN entities pe ON pe.id = lp.entity_id AND pe.kind = '`+string(domain.EntityProcesso)+`'
		), process_max AS (
			SELECT DISTINCT ON (key) key, record_id, value FROM processes ORDER BY key, value DESC, record_id
		)
		SELECT (SELECT count(*) FROM process_max),
		       coalesce((SELECT sum(value) FROM (SELECT DISTINCT ON (record_id) value FROM process_max ORDER BY record_id) s), 0)`,
		entityID, domain.RecordAct, source).Scan(&report.ProcessCount, &report.ProcessSumCents)
}
