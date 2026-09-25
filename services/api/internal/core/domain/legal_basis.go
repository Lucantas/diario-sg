package domain

import (
	"regexp"
	"slices"
	"strings"
)

type LegalBasis string

const (
	Art24I    LegalBasis = "art24:I"
	Art24II   LegalBasis = "art24:II"
	Art24IV   LegalBasis = "art24:IV"
	Art75I    LegalBasis = "art75:I"
	Art75II   LegalBasis = "art75:II"
	Art75VIII LegalBasis = "art75:VIII"
)

var knownLegalBases = []LegalBasis{Art24I, Art24II, Art24IV, Art75I, Art75II, Art75VIII}

var (
	articleThenIncisoRe = regexp.MustCompile(`(?i)art(?:igo)?\.?\s*(24|75)\s*,?\s*(?:caput\s*,?\s*)?(?:inciso\s*|inc\.\s*)?([IVX]+)(?:[^IVXa-zà-ú]|$)`)
	incisoThenArticleRe = regexp.MustCompile(`(?i)inciso\s*([IVX]+)\s*,?\s*d[oa]\s*art(?:igo)?\.?\s*(24|75)(?:\D|$)`)
)

func LegalBasisOf(body string) []LegalBasis {
	var out []LegalBasis
	add := func(article, inciso string) {
		b := LegalBasis("art" + article + ":" + strings.ToUpper(inciso))
		if slices.Contains(knownLegalBases, b) && !slices.Contains(out, b) {
			out = append(out, b)
		}
	}
	for _, m := range articleThenIncisoRe.FindAllStringSubmatch(body, -1) {
		add(m[1], m[2])
	}
	for _, m := range incisoThenArticleRe.FindAllStringSubmatch(body, -1) {
		add(m[2], m[1])
	}
	slices.Sort(out)
	return out
}

func citesAny(body string, bases ...LegalBasis) bool {
	for _, b := range LegalBasisOf(body) {
		if slices.Contains(bases, b) {
			return true
		}
	}
	return false
}
