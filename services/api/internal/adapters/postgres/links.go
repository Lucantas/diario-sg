package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func linkActs(ctx context.Context, tx *sql.Tx, gazetteID string) error {
	_, err := tx.ExecContext(ctx, `SELECT link_diario_acts($1)`, gazetteID)
	return err
}

func unlinkActs(ctx context.Context, tx *sql.Tx, gazetteID string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM entity_links
		WHERE source = $2 AND record_kind = $3
		  AND record_id IN (SELECT id::text FROM acts WHERE gazette_id = $1)`,
		gazetteID, domain.SourceDiarioPrefeitura, domain.RecordAct)
	return err
}

type LinkRepo struct{ db *sql.DB }

func NewLinkRepo(db *sql.DB) *LinkRepo { return &LinkRepo{db: db} }

const weakestCertainty = `(ARRAY['fraca', 'forte', 'exata'])[min(array_position(ARRAY['fraca', 'forte', 'exata'], l.certainty))]`

func (r *LinkRepo) ReportByKey(ctx context.Context, kind domain.EntityKind, key string) (domain.EntityReport, error) {
	report := domain.EntityReport{Kind: kind, Key: key, CountByType: map[domain.ActType]int{}}
	var entityID, certainty string
	err := r.db.QueryRowContext(ctx, `
		SELECT e.id, `+weakestCertainty+`
		FROM entities e JOIN entity_links l ON l.entity_id = e.id AND l.source = $3 AND l.record_kind = $4
		WHERE e.kind = $1 AND e.key = $2
		GROUP BY e.id`, string(kind), key, domain.SourceDiarioPrefeitura, domain.RecordAct).Scan(&entityID, &certainty)
	if errors.Is(err, sql.ErrNoRows) {
		return report, nil
	}
	if err != nil {
		return report, err
	}
	report.Certainty = domain.Certainty(certainty)
	if err := r.linkedActs(ctx, entityID, &report); err != nil {
		return report, err
	}
	if err := r.linkedTypes(ctx, entityID, &report); err != nil {
		return report, err
	}
	err = r.db.QueryRowContext(ctx, `
		SELECT coalesce(sum(v.normalized::bigint), 0)
		FROM entity_links l JOIN act_entities v ON v.act_id = l.record_id::uuid AND v.kind = 'valor'
		WHERE l.entity_id = $1 AND l.source = $2 AND l.record_kind = $3`,
		entityID, domain.SourceDiarioPrefeitura, domain.RecordAct).Scan(&report.TotalCents)
	return report, err
}

func (r *LinkRepo) linkedActs(ctx context.Context, entityID string, report *domain.EntityReport) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, a.organ, coalesce(a.page_start, 0), coalesce(a.page_end, 0),
		       g.edition_number, g.published_at, g.is_extra, g.source_url, g.checksum,
		       substr(a.body, greatest(position(l.evidence IN a.body) - $4, 1), 2 * $4 + length(l.evidence)), l.evidence
		FROM entity_links l
		JOIN acts a ON a.id = l.record_id::uuid
		JOIN gazettes g ON g.id = a.gazette_id
		WHERE l.entity_id = $1 AND l.source = $2 AND l.record_kind = $3
		ORDER BY g.published_at DESC, a.position
		LIMIT $5`, entityID, domain.SourceDiarioPrefeitura, domain.RecordAct, snippetRadius, reportActsLimit)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var h domain.ActHit
		var typ, evidence string
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Position, &h.Organ, &h.PageStart, &h.PageEnd,
			&h.EditionNumber, &h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Checksum, &h.Snippet, &evidence); err != nil {
			return err
		}
		h.Type = domain.ActType(typ)
		h.Snippet = highlightFallback(h.Snippet, evidence)
		report.Acts = append(report.Acts, h)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return markTitleOnly(ctx, r.db, report.Acts)
}

func (r *LinkRepo) linkedTypes(ctx context.Context, entityID string, report *domain.EntityReport) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.type, count(*)
		FROM entity_links l JOIN acts a ON a.id = l.record_id::uuid
		WHERE l.entity_id = $1 AND l.source = $2 AND l.record_kind = $3
		GROUP BY a.type`, entityID, domain.SourceDiarioPrefeitura, domain.RecordAct)
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
