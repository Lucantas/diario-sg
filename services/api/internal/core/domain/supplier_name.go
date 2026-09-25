package domain

import (
	"regexp"
	"strings"
)

const (
	supplierNameWindow = 160
	partiesWindow      = 220
	minSupplierName    = 4
)

var (
	supplierLeadRe     = regexp.MustCompile(`(?i)(?:em\s+favor\s+d[ao]s?|favorecid[ao]s?\s*:?|contratad[ao]\s*:|empresa\s+contratada\s*:?)\s+(?:(?:a\s+)?empresa\s+)?`)
	partiesLeadRe      = regexp.MustCompile(`(?i)partes\s*:\s*`)
	partiesStrongSepRe = regexp.MustCompile(`(?i)\s(?:e\s+a\s+empresa|x)\s+`)
	partiesWeakSepRe   = regexp.MustCompile(`(?i)\se\s+(?:(?:a|o)\s+)?(?:empresa\s+)?`)
	supplierEndRe      = regexp.MustCompile(`\s*(?:,|;|\s[-–]\s|\.\s*\p{Lu}\p{Ll}|\.\s*$|\s(?i:inscrit|cnpj|pessoa\s+jur|com\s+sede|objeto|cujo|para|no\s+valor|valor|referente|processo|situad|estabelecid|neste\s+ato|representad|doravante)\b)`)
	publicPartyRe      = regexp.MustCompile(`(?i)munic[íi]pi|prefeitura|secretaria|funda[çc][ãa]o\s+municipal|procuradoria|fundo\s+municipal|instituto\s+de\s+previd|^lazer\b`)
	supplierLetterRe   = regexp.MustCompile(`\pL{3}`)
	nameKeyNoiseRe     = regexp.MustCompile(`[^a-z0-9 ]+`)
	nameKeySuffixRe    = regexp.MustCompile(`(?:\s(?:ltda|eireli|epp|me|s a|sa|s s|ss|limitada|microempresa|cia|e cia|sociedade simples|sociedade unipessoal))+$`)
	accentFolder       = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "ê", "e", "è", "e", "í", "i", "ì", "i", "ó", "o", "ô", "o", "õ", "o", "ò", "o", "ú", "u", "ü", "u", "ç", "c")
)

func SupplierNameOf(body string) string {
	text := strings.Join(strings.Fields(body), " ")
	type lead struct{ start, end int }
	var leads []lead
	for _, m := range supplierLeadRe.FindAllStringIndex(text, -1) {
		leads = append(leads, lead{m[0], m[1]})
	}
	for _, m := range partiesLeadRe.FindAllStringIndex(text, -1) {
		if end, ok := afterPartiesSeparator(text, m[1]); ok {
			leads = append(leads, lead{m[0], end})
		}
	}
	best, bestAt := "", len(text)+1
	for _, l := range leads {
		if name := supplierNameAt(text, l.end); name != "" && l.start < bestAt {
			best, bestAt = name, l.start
		}
	}
	return best
}

func afterPartiesSeparator(text string, from int) (int, bool) {
	window := text[from:min(len(text), from+partiesWindow)]
	for _, re := range []*regexp.Regexp{partiesStrongSepRe, partiesWeakSepRe} {
		if loc := re.FindStringIndex(window); loc != nil {
			return from + loc[1], true
		}
	}
	return 0, false
}

func supplierNameAt(text string, from int) string {
	rest := text[from:min(len(text), from+supplierNameWindow)]
	if loc := supplierEndRe.FindStringIndex(rest); loc != nil {
		rest = rest[:loc[0]]
	}
	name := strings.Trim(rest, " \"“”.,;-–")
	if len([]rune(name)) < minSupplierName || !supplierLetterRe.MatchString(name) || publicPartyRe.MatchString(name) {
		return ""
	}
	return name
}

func SupplierNameKey(name string) string {
	key := accentFolder.Replace(strings.ToLower(name))
	key = strings.Join(strings.Fields(nameKeyNoiseRe.ReplaceAllString(key, " ")), " ")
	return strings.TrimSpace(nameKeySuffixRe.ReplaceAllString(" "+key, ""))
}
