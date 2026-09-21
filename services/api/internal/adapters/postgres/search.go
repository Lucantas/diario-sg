package postgres

import "strings"

// Configuração de busca criada na migration 002: português + unaccent.
const tsConfig = "portuguese_unaccent"

// Marcadores de destaque: evitamos HTML para não abrir brecha de XSS no front.
const headlineOpts = `StartSel=⟦, StopSel=⟧, MaxFragments=2, MaxWords=35, MinWords=12`

// likePattern transforma a busca do usuário em padrão ILIKE de substring,
// escapando os curingas do LIKE.
func likePattern(q string) string {
	q = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
	return "%" + q + "%"
}

// Um ato casa quando a busca textual (stemming, sem acentos) bate OU quando
// o termo aparece literalmente no corpo (também sem acentos). O segundo caso
// cobre nomes parciais, CNPJ e números de contrato que o tokenizador de
// texto não trata bem.
const matchClause = `(a.search @@ q OR unaccent_immutable(a.body) ILIKE unaccent_immutable($LIKE))`
