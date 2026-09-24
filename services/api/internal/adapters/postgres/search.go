package postgres

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const tsConfig = "portuguese_unaccent"

const headlineOpts = `StartSel=⟦, StopSel=⟧, MaxFragments=2, MaxWords=35, MinWords=12`

func phraseOf(q string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(q), `"`))
}

func likePattern(q string) string {
	q = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(phraseOf(q))
	return "%" + q + "%"
}

const exactPhrase = `unaccent_immutable(a.body) ILIKE unaccent_immutable($LIKE)`

func exactPhraseExpr(likeParam string) string {
	return strings.Replace(exactPhrase, "$LIKE", likeParam, 1)
}

func exactPhraseFor(likeParam string) string {
	return `CROSS JOIN LATERAL (SELECT ` + exactPhraseExpr(likeParam) + ` AS exact) x`
}

const matchClause = `(a.search @@ q
		  OR (($Q ~ '[0-9]' OR length($Q) >= ` + minSubstringRunes + `)
		      AND unaccent_immutable(a.body) ILIKE unaccent_immutable($LIKE)))`

const minSubstringRunes = "8"

func matchFor(queryParam, likeParam string) string {
	return strings.NewReplacer("$Q", queryParam, "$LIKE", likeParam).Replace(matchClause)
}

const cnpjsSubquery = `(SELECT coalesce(array_agg(DISTINCT e.normalized), '{}')
		        FROM act_entities e WHERE e.act_id = a.id AND e.kind = 'cnpj')`

const valuesSubquery = `(SELECT coalesce(array_agg(v.normalized::bigint ORDER BY v.normalized::bigint DESC), '{}')
		        FROM act_entities v WHERE v.act_id = a.id AND v.kind = 'valor')`

const mentionsSubquery = `(SELECT coalesce(array_agg(DISTINCT m.kind || '|' || entity_key(m.kind, m.normalized) || '|' || m.value), '{}')
		        FROM act_entities m WHERE m.act_id = a.id AND m.kind IN ('processo', 'contrato'))`

func parseMentions(raw []string) []domain.EntityMention {
	seen := map[string]bool{}
	var out []domain.EntityMention
	for _, r := range raw {
		parts := strings.SplitN(r, "|", 3)
		if len(parts) != 3 || seen[parts[0]+parts[1]] {
			continue
		}
		seen[parts[0]+parts[1]] = true
		kind := domain.EntityKind(parts[0])
		out = append(out, domain.EntityMention{Kind: kind, Key: parts[1], Label: domain.EntityLabel(kind, parts[2])})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind > out[j].Kind
		}
		return out[i].Key < out[j].Key
	})
	return out
}

const (
	relevanceOrder = `ORDER BY x.exact DESC,
		         CASE WHEN $1 = '' OR x.exact THEN 0 ELSE ts_rank(a.search, q) END DESC,
		         g.published_at DESC, a.position`
	recentOrder = `ORDER BY g.published_at DESC, g.source_url DESC, a.position`

	matchedRelevanceOrder = `ORDER BY exact DESC, rank DESC, published_at DESC, position`
	matchedRecentOrder    = `ORDER BY published_at DESC, source_url DESC, position`

	pageRelevanceOrder = `ORDER BY p.exact DESC, p.rank DESC, p.published_at DESC, p.position`
	pageRecentOrder    = `ORDER BY p.published_at DESC, p.source_url DESC, p.position`
)

func orderSQL(f domain.ActFilter) string {
	if f.Recent {
		return recentOrder
	}
	return relevanceOrder
}

func matchedOrderSQL(f domain.ActFilter) string {
	if f.Recent {
		return matchedRecentOrder
	}
	return matchedRelevanceOrder
}

func pageOrderSQL(f domain.ActFilter) string {
	if f.Recent {
		return pageRecentOrder
	}
	return pageRelevanceOrder
}

func markBodyFacts(ctx context.Context, db *sql.DB, hits []domain.ActHit) error {
	if len(hits) == 0 {
		return nil
	}
	ids := make([]string, len(hits))
	for i, h := range hits {
		ids[i] = h.ID
	}
	rows, err := db.QueryContext(ctx, `SELECT id, title, body FROM acts WHERE id = ANY($1::uuid[])`, pq.Array(ids))
	if err != nil {
		return err
	}
	defer rows.Close()
	facts := map[string]domain.WarningFacts{}
	for rows.Next() {
		var a domain.Act
		if err := rows.Scan(&a.ID, &a.Title, &a.Body); err != nil {
			return err
		}
		facts[a.ID] = domain.WarningFactsOf(a)
	}
	for i := range hits {
		f := facts[hits[i].ID]
		hits[i].TitleOnly, hits[i].Signatures = f.TitleOnly, f.Signatures
	}
	return rows.Err()
}
