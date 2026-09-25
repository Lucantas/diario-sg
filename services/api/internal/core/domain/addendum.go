package domain

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	addendumHeadRunes   = 300
	basisPointsPerPoint = 100
)

var (
	addendumTitleRe      = regexp.MustCompile(`(?i)aditiv`)
	increaseThenPctRe    = regexp.MustCompile(`(?i)acr[ée]scimo(?:[^.;%]|\.\d){0,200}?(\d{1,3})(?:,(\d{1,2}))?\d*\s?%`)
	pctThenIncreaseRe    = regexp.MustCompile(`(?i)(\d{1,3})(?:,(\d{1,2}))?\d*\s?%(?:[^.;]|\.\d){0,40}?acr[ée]scid`)
	readjustmentRe       = regexp.MustCompile(`(?i)reajust|repactua|reequil|revis[ãa]o\s+de\s+pre[çc]o|ipca|igp|[íi]ndice`)
	quantitativeRe       = regexp.MustCompile(`(?i)quantitativ|qualitativ`)
	legalCeilingBeforeRe = regexp.MustCompile(`(?i)(?:at[ée]|limite\s+de|limites?\s+legal)\s*$`)
	rectificationRe      = regexp.MustCompile(`(?i)onde\s+se\s+l[êe]|^\W*(?:extrato\s+d[aeo]\s+)?retifica`)
	renovationRe         = regexp.MustCompile(`(?i)\breformas?\b`)
	ordinalRe            = regexp.MustCompile(`(?i)\b(primeiro|segundo|terceiro|quarto|quinto|sexto|s[ée]timo|oitavo|nono|d[ée]cimo|\d{1,2})\s*(?:º|°|o\.?)?\s*termo\s+aditivo`)
)

var ordinalWords = map[string]int{
	"primeiro": 1, "segundo": 2, "terceiro": 3, "quarto": 4, "quinto": 5, "sexto": 6,
	"sétimo": 7, "setimo": 7, "oitavo": 8, "nono": 9, "décimo": 10, "decimo": 10,
}

func DeclaredIncreaseBasisPoints(t ActType, title, body string) int {
	text := strings.Join(strings.Fields(body), " ")
	head := title + " " + firstRunes(text, addendumHeadRunes)
	if t != ActAditivo && !addendumTitleRe.MatchString(title) {
		return 0
	}
	if rectificationRe.MatchString(text) || (readjustmentRe.MatchString(head) && !quantitativeRe.MatchString(head)) {
		return 0
	}
	for _, re := range []*regexp.Regexp{increaseThenPctRe, pctThenIncreaseRe} {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			if bp, ok := increaseAt(text, m); ok {
				return bp
			}
		}
	}
	return 0
}

func increaseAt(text string, m []int) (int, bool) {
	if readjustmentRe.MatchString(text[m[0]:m[1]]) || legalCeilingBeforeRe.MatchString(text[:m[2]]) {
		return 0, false
	}
	whole, err := strconv.Atoi(text[m[2]:m[3]])
	if err != nil {
		return 0, false
	}
	fraction := 0
	if m[4] >= 0 {
		digits := (text[m[4]:m[5]] + "0")[:2]
		fraction, _ = strconv.Atoi(digits)
	}
	return whole*basisPointsPerPoint + fraction, true
}

func AddendumOrdinal(title, body string) int {
	m := ordinalRe.FindStringSubmatch(title + " " + firstRunes(body, addendumHeadRunes))
	if m == nil {
		return 0
	}
	word := strings.ToLower(m[1])
	if n, err := strconv.Atoi(word); err == nil {
		return n
	}
	return ordinalWords[word]
}

func MentionsRenovation(body string) bool { return renovationRe.MatchString(body) }

func firstRunes(s string, n int) string {
	r := []rune(s)
	return string(r[:min(len(r), n)])
}
