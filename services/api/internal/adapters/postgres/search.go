package postgres

import (
	"strings"

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

const exactPhraseJoin = `CROSS JOIN LATERAL (SELECT unaccent_immutable(a.body) ILIKE unaccent_immutable($LIKE) AS exact) x`

func exactPhraseFor(likeParam string) string {
	return strings.Replace(exactPhraseJoin, "$LIKE", likeParam, 1)
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

const (
	relevanceOrder = `ORDER BY x.exact DESC,
		         CASE WHEN $1 = '' OR x.exact THEN 0 ELSE ts_rank(a.search, q) END DESC,
		         g.published_at DESC, a.position`
	recentOrder = `ORDER BY g.published_at DESC, g.source_url DESC, a.position`
)

func orderSQL(f domain.ActFilter) string {
	if f.Recent {
		return recentOrder
	}
	return relevanceOrder
}

const titleOnlyExpr = `btrim(a.body) = btrim(a.title)`
