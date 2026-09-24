package postgres

import (
	"context"
	"strconv"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func (r *ActRepo) Group(ctx context.Context, q domain.GroupQuery) (domain.ActGroups, error) {
	keySQL, args := groupKeySQL(q)
	where, extra := filterSQL(q.Filter, len(args)+1)
	args = append(args, extra...)
	rows, err := r.db.QueryContext(ctx, `
		WITH matched AS MATERIALIZED (`+matchedSQL("$1", "$3", where)+`),
		keyed AS (`+keySQL+`),
		valued AS (
			SELECT k.key, k.id, k.published_at,
			       (SELECT max(v.normalized::bigint) FROM act_entities v WHERE v.act_id = k.id AND v.kind = 'valor') AS max_value
			FROM keyed k
		)
		SELECT key, count(DISTINCT id), min(published_at), max(published_at), coalesce(max(max_value), 0),
		       (array_agg(id::text ORDER BY published_at DESC, id))[1:`+strconv.Itoa(domain.GroupExamples)+`]
		FROM valued
		GROUP BY key
		ORDER BY count(DISTINCT id) DESC, key
		LIMIT $2`, args...)
	if err != nil {
		return domain.ActGroups{}, err
	}
	defer rows.Close()

	var out domain.ActGroups
	var exampleIDs []string
	examplesOf := map[string][]string{}
	for rows.Next() {
		var g domain.ActGroup
		var ids []string
		if err := rows.Scan(&g.Key, &g.Acts, &g.First, &g.Last, &g.MaxValueCents, pq.Array(&ids)); err != nil {
			return out, err
		}
		examplesOf[g.Key] = ids
		exampleIDs = append(exampleIDs, ids...)
		out.Groups = append(out.Groups, g)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if out.MatchedActs, err = r.countMatchedIn(ctx, q.Filter); err != nil {
		return out, err
	}
	refs, err := r.actRefs(ctx, exampleIDs)
	if err != nil {
		return out, err
	}
	for i, g := range out.Groups {
		for _, id := range examplesOf[g.Key] {
			out.Groups[i].Examples = append(out.Groups[i].Examples, refs[id])
		}
	}
	return out, nil
}

func groupKeySQL(q domain.GroupQuery) (string, []any) {
	base := []any{q.Filter.Query, q.Limit, likePattern(q.Filter.Query)}
	switch q.By {
	case domain.GroupByOrgan:
		return `SELECT coalesce(a.organ, '') AS key, m.id, m.published_at FROM matched m JOIN acts a ON a.id = m.id`, base
	case domain.GroupByType:
		return `SELECT a.type AS key, m.id, m.published_at FROM matched m JOIN acts a ON a.id = m.id`, base
	}
	kind, _ := q.By.EntityKind()
	return `SELECT e.key, m.id, m.published_at
		FROM matched m
		JOIN entity_links l ON l.record_kind = '` + domain.RecordAct + `' AND l.record_id = m.id::text
		JOIN entities e ON e.id = l.entity_id AND e.kind = $4
		WHERE NOT (e.key = ANY($5) OR left(e.key, 8) = ANY($5))`,
		append(base, string(kind), pq.StringArray(q.ExcludedKeys()))
}

func (r *ActRepo) countMatchedIn(ctx context.Context, f domain.ActFilter) (int, error) {
	where, extra := filterSQL(f, 3)
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM (`+matchedSQL("$1", "$2", where)+`) m`,
		append([]any{f.Query, likePattern(f.Query)}, extra...)...).Scan(&total)
	return total, err
}

func (r *ActRepo) actRefs(ctx context.Context, ids []string) (map[string]domain.ActRef, error) {
	refs := map[string]domain.ActRef{}
	if len(ids) == 0 {
		return refs, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.gazette_id, a.position, a.title, g.source, g.published_at
		FROM acts a JOIN gazettes g ON g.id = a.gazette_id
		WHERE a.id = ANY($1::uuid[])`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var ref domain.ActRef
		if err := rows.Scan(&id, &ref.GazetteID, &ref.Position, &ref.Title, &ref.Source, &ref.PublishedAt); err != nil {
			return nil, err
		}
		refs[id] = ref
	}
	return refs, rows.Err()
}
