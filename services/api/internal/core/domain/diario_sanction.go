package domain

import (
	"regexp"
	"strings"
)

type DiarioSanctionKind string

const (
	SanctionWarning       DiarioSanctionKind = "advertencia"
	SanctionFine          DiarioSanctionKind = "multa"
	SanctionSuspension    DiarioSanctionKind = "suspensao"
	SanctionDebarment     DiarioSanctionKind = "impedimento"
	SanctionIneligibility DiarioSanctionKind = "inidoneidade"
)

type DiarioSanction struct {
	Kind DiarioSanctionKind
	Act  ActHit
}

type SanctionCandidate struct {
	ActID string
	Title string
	Body  string
}

var (
	sanctionTitleRe    = regexp.MustCompile(`\b(advertencia|multa|sancao|penalidade)\b`)
	sanctionAmnestyRe  = regexp.MustCompile(`\banistia\b`)
	sanctionDecisionRe = regexp.MustCompile(`\b(aplico|aplica|aplicar|aplicando|aplica-se|aplicase|decide aplicar|impoe|imponho|impor)\b[^.;]{0,60}?` +
		`\b(penalidade|sancao|pena|multa)\b[^.;]{0,40}?\bde (multa|advertencia|suspensao|impedimento|declaracao de inidoneidade|inidoneidade)`)
	sanctionResolveRe = regexp.MustCompile(`\b(resolve|decide|decido)\b`)
	sanctionKindOrder = []struct {
		word string
		kind DiarioSanctionKind
	}{{"inidoneidade", SanctionIneligibility}, {"impedimento", SanctionDebarment}, {"suspensao", SanctionSuspension},
		{"multa", SanctionFine}, {"advertencia", SanctionWarning}, {"advertida", SanctionWarning}}
)

func ClassifyDiarioSanction(title, body string) (DiarioSanctionKind, bool) {
	t, b := foldAccents(title), foldAccents(body)
	if sanctionTitleRe.MatchString(t) && !sanctionAmnestyRe.MatchString(t) {
		if kind, ok := sanctionKindIn(t); ok {
			return kind, true
		}
		if m := sanctionDecision(b); m != "" {
			return sanctionKindIn(m)
		}
		return sanctionKindIn(firstChars(b, sanctionBodyWindow))
	}
	if m := sanctionDecision(b); m != "" {
		return sanctionKindIn(m)
	}
	return "", false
}

func sanctionDecision(body string) string {
	for _, loc := range sanctionDecisionRe.FindAllStringSubmatchIndex(body, -1) {
		verb := body[loc[2]:loc[3]]
		if verb != "aplicar" && verb != "impor" {
			return body[loc[0]:loc[1]]
		}
		if sanctionResolveRe.MatchString(body[max(0, loc[0]-sanctionResolveWindow):loc[0]]) {
			return body[loc[0]:loc[1]]
		}
	}
	return ""
}

const (
	sanctionBodyWindow    = 600
	sanctionResolveWindow = 40
)

func sanctionKindIn(text string) (DiarioSanctionKind, bool) {
	for _, k := range sanctionKindOrder {
		if strings.Contains(text, k.word) {
			return k.kind, true
		}
	}
	return "", false
}

func firstChars(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
