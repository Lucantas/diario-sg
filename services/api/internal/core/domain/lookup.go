package domain

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	minLookupLength  = 3
	maxLookupLength  = 100
	minNameLetters   = 3
	SuggestNameLimit = 5
)

type Lookup struct {
	CNPJ          string
	Processo      string
	ProcessoLabel string
	Contrato      string
	ContratoLabel string
	Name          string
}

type Suggestion struct {
	Kind  EntityKind
	Key   string
	Label string
	Name  string
	Acts  int
}

var (
	lookupKeywordRe = regexp.MustCompile(`(?i)^(processo|contrato)\s+(?:n\.?\s*[º°o]\.?\s*)?(.*)$`)
	processoShapeRe = regexp.MustCompile(`^[\d./-]+$`)
	likeEscaper     = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
)

func ParseLookup(q string) Lookup {
	q = strings.TrimSpace(q)
	if n := utf8.RuneCountInString(q); n < minLookupLength || n > maxLookupLength {
		return Lookup{}
	}
	if cnpj, ok := NormalizeCNPJ(q); ok {
		return Lookup{CNPJ: cnpj}
	}
	if m := lookupKeywordRe.FindStringSubmatch(q); m != nil {
		return numberLookup(EntityKind(strings.ToLower(m[1])), strings.TrimSpace(m[2]))
	}
	if startsWithDigit(q) {
		return numberLookup("", q)
	}
	if countLetters(q) >= minNameLetters {
		return Lookup{Name: q}
	}
	return Lookup{}
}

func numberLookup(only EntityKind, number string) Lookup {
	var l Lookup
	if only != EntityContrato && processoShapeRe.MatchString(number) {
		if key, err := ParseEntityInput(EntityProcesso, number); err == nil {
			l.Processo, l.ProcessoLabel = key, number
		}
	}
	if only != EntityProcesso {
		if key, err := ParseEntityInput(EntityContrato, number); err == nil {
			l.Contrato, l.ContratoLabel = key, strings.ToUpper(number)
		}
	}
	return l
}

func ContainsPattern(text string) string { return "%" + likeEscaper.Replace(text) + "%" }

func startsWithDigit(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsDigit(r)
}

func countLetters(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}
