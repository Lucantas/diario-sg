package postgres

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const (
	snippetRadius = 120

	reportActsLimit = 100
)

func (r *ActRepo) ReportByEntity(ctx context.Context, kind domain.EntityKind, normalized string) (domain.CompanyReport, error) {
	report := domain.CompanyReport{CNPJ: normalized, CountByType: map[domain.ActType]int{}}
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.type, a.title, a.position, a.organ, coalesce(a.page_start, 0), coalesce(a.page_end, 0),
		       g.edition_number, g.published_at, g.is_extra, g.source_url, g.checksum,
		       substr(a.body, greatest(position(e.value IN a.body) - $3, 1), 2 * $3 + length(e.value)), e.value
		FROM act_entities e
		JOIN acts a ON a.id = e.act_id
		JOIN gazettes g ON g.id = a.gazette_id
		WHERE e.kind = $1 AND e.normalized = $2
		ORDER BY g.published_at DESC, a.position
		LIMIT $4`, string(kind), normalized, snippetRadius, reportActsLimit)
	if err != nil {
		return report, err
	}
	defer rows.Close()
	for rows.Next() {
		var h domain.ActHit
		var typ, value string
		if err := rows.Scan(&h.ID, &h.GazetteID, &typ, &h.Title, &h.Position, &h.Organ, &h.PageStart, &h.PageEnd,
			&h.EditionNumber, &h.PublishedAt, &h.IsExtra, &h.SourceURL, &h.Checksum, &h.Snippet, &value); err != nil {
			return report, err
		}
		h.Type = domain.ActType(typ)
		h.Snippet = highlightFallback(h.Snippet, value)
		report.Acts = append(report.Acts, h)
	}
	if err := rows.Err(); err != nil {
		return report, err
	}

	types, err := r.db.QueryContext(ctx, `
		SELECT a.type, count(*)
		FROM act_entities e JOIN acts a ON a.id = e.act_id
		WHERE e.kind = $1 AND e.normalized = $2
		GROUP BY a.type`, string(kind), normalized)
	if err != nil {
		return report, err
	}
	defer types.Close()
	for types.Next() {
		var typ string
		var n int
		if err := types.Scan(&typ, &n); err != nil {
			return report, err
		}
		report.CountByType[domain.ActType(typ)] = n
	}
	if err := types.Err(); err != nil {
		return report, err
	}

	err = r.db.QueryRowContext(ctx, `
		SELECT coalesce(sum(v.normalized::bigint), 0)
		FROM act_entities e
		JOIN act_entities v ON v.act_id = e.act_id AND v.kind = 'valor'
		WHERE e.kind = $1 AND e.normalized = $2`, string(kind), normalized).Scan(&report.TotalCents)
	return report, err
}

func (r *ActRepo) CountByMonth(ctx context.Context, f domain.ActFilter) ([]domain.MonthCount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT date_trunc('month', g.published_at)::date AS month, count(*)
		FROM acts a
		JOIN gazettes g ON g.id = a.gazette_id
		CROSS JOIN websearch_to_tsquery('`+tsConfig+`', $1) q
		WHERE ($1 = '' OR `+matchFor("$1", "$5")+`)
		  AND ($2 = '' OR a.type = $2)
		  AND ($3::date IS NULL OR g.published_at >= $3::date)
		  AND ($4::date IS NULL OR g.published_at <= $4::date)
		  AND ($6 = '' OR a.organ = $6)
		GROUP BY 1 ORDER BY 1`, f.Query, string(f.Type), nullableDate(f.From), nullableDate(f.To), likePattern(f.Query), f.Organ)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MonthCount
	for rows.Next() {
		var m domain.MonthCount
		if err := rows.Scan(&m.Month, &m.Count); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *ActRepo) CountByOrgan(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT organ, count(*) FROM acts WHERE organ <> '' GROUP BY organ`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var organ string
		var n int
		if err := rows.Scan(&organ, &n); err != nil {
			return nil, err
		}
		out[organ] = n
	}
	return out, rows.Err()
}
