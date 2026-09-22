package postgres

import "strings"

// Configuração de busca criada na migration 002: português + unaccent.
const tsConfig = "portuguese_unaccent"

// Marcadores de destaque: evitamos HTML para não abrir brecha de XSS no front.
const headlineOpts = `StartSel=⟦, StopSel=⟧, MaxFragments=2, MaxWords=35, MinWords=12`

// phraseOf tira as aspas que o usuário usa para pedir frase exata: o
// websearch_to_tsquery as entende, mas a comparação literal do corpo não.
func phraseOf(q string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(q), `"`))
}

// likePattern transforma a busca do usuário em padrão ILIKE de substring,
// escapando os curingas do LIKE.
func likePattern(q string) string {
	q = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(phraseOf(q))
	return "%" + q + "%"
}

// Ordem dos resultados: quem traz a frase exata (sem acento, sem caixa) vem
// antes, do mais recente ao mais antigo; o resto segue o ts_rank, que sozinho
// favorece listas longas de nomes com as palavras soltas.
const exactPhraseJoin = `CROSS JOIN LATERAL (SELECT unaccent_immutable(a.body) ILIKE unaccent_immutable($LIKE) AS exact) x`

func exactPhraseFor(likeParam string) string {
	return strings.Replace(exactPhraseJoin, "$LIKE", likeParam, 1)
}

// Um ato casa quando a busca textual (stemming, sem acentos) bate OU quando
// o termo aparece literalmente no corpo (também sem acentos). O segundo caso
// cobre CNPJ, números de contrato/processo e nomes longos que o tokenizador
// não trata bem. Termos curtos e sem dígito ("sus", "lei") ficam só na busca
// textual: como substring casariam com quase tudo e inundariam os alertas.
const matchClause = `(a.search @@ q
		  OR (($Q ~ '[0-9]' OR length($Q) >= ` + minSubstringRunes + `)
		      AND unaccent_immutable(a.body) ILIKE unaccent_immutable($LIKE)))`

const minSubstringRunes = "8"

// matchFor instancia matchClause com os índices do termo e do padrão LIKE.
func matchFor(queryParam, likeParam string) string {
	return strings.NewReplacer("$Q", queryParam, "$LIKE", likeParam).Replace(matchClause)
}

// CNPJs citados no ato (só dígitos), para o front linkar a página da empresa.
const cnpjsSubquery = `(SELECT coalesce(array_agg(DISTINCT e.normalized), '{}')
		        FROM act_entities e WHERE e.act_id = a.id AND e.kind = 'cnpj')`
