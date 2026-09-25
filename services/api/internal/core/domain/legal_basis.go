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

const (
	lawBeforeWindow  = 60
	lawAfterWindow   = 100
	vetoBeforeWindow = 80
)

var (
	procurementLawRe    = regexp.MustCompile(`(?i)8\.?666|14\.?133|lei\s+(?:federal\s+)?de\s+licita[çc][õo]es`)
	legalBasisVetoRe    = regexp.MustCompile(`(?i)vedad|hip[óo]tese|recontrata`)
	incisoListRe        = `(?:incisos?\s*|incs?\.\s*)?[IVX]+(?:\s*(?:,|e|ou)\s*(?:inc(?:iso|\.)?\s*)?[IVX]+\b)*`
	articleThenIncisoRe = regexp.MustCompile(`(?i)art(?:igo)?\.?\s*(24|75)\s*,?\s*(?:caput\s*,?\s*)?(` + incisoListRe + `)(?:[^IVXa-zà-ú]|$)`)
	incisoThenArticleRe = regexp.MustCompile(`(?i)incisos?\s*(` + incisoListRe + `)\s*,?\s*d[oa]\s*art(?:igo)?\.?\s*(24|75)(?:\D|$)`)
	romanNumeralRe      = regexp.MustCompile(`(?i)\b[IVX]+\b`)
)

func LegalBasisOf(body string) []LegalBasis {
	var out []LegalBasis
	add := func(m []int, article, incisos string) {
		if !citesProcurementLaw(body, m) {
			return
		}
		for _, inciso := range romanNumeralRe.FindAllString(incisos, -1) {
			b := LegalBasis("art" + article + ":" + strings.ToUpper(inciso))
			if slices.Contains(knownLegalBases, b) && !slices.Contains(out, b) {
				out = append(out, b)
			}
		}
	}
	for _, m := range articleThenIncisoRe.FindAllStringSubmatchIndex(body, -1) {
		add(m, body[m[2]:m[3]], body[m[4]:m[5]])
	}
	for _, m := range incisoThenArticleRe.FindAllStringSubmatchIndex(body, -1) {
		add(m, body[m[4]:m[5]], body[m[2]:m[3]])
	}
	slices.Sort(out)
	return out
}

func citesProcurementLaw(body string, m []int) bool {
	before := body[max(0, m[0]-lawBeforeWindow):m[0]]
	after := body[m[1]:min(len(body), m[1]+lawAfterWindow)]
	vetoZone := body[max(0, m[0]-vetoBeforeWindow):m[0]]
	return !legalBasisVetoRe.MatchString(vetoZone) && procurementLawRe.MatchString(before+" "+after)
}

func citesAny(body string, bases ...LegalBasis) bool {
	for _, b := range LegalBasisOf(body) {
		if slices.Contains(bases, b) {
			return true
		}
	}
	return false
}
