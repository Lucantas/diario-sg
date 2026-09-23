package postgres

import "strings"

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
