package domain

import (
	"regexp"
	"strings"
)

type Certainty string

const (
	CertaintyExact  Certainty = "exata"
	CertaintyStrong Certainty = "forte"
	CertaintyWeak   Certainty = "fraca"
)

const (
	RecordAct     = "ato"
	RoleMentioned = "mencionado"

	minProcessoDigits = 5
)

var (
	contratoShapeRe     = regexp.MustCompile(`^\d{1,5}(?:/[A-Z]{2,10})?/\d{4}(?:/[A-Z]{2,10})?$`)
	contratoLeadZerosRe = regexp.MustCompile(`^0+(\d)`)
	nonDigitRe          = regexp.MustCompile(`\D`)
)

func IsContratoShape(s string) bool { return contratoShapeRe.MatchString(s) }

func IsLinkedKind(kind EntityKind) bool {
	return kind == EntityCNPJ || kind == EntityProcesso || kind == EntityContrato
}

func EntityKey(kind EntityKind, normalized string) string {
	if kind == EntityContrato {
		return contratoLeadZerosRe.ReplaceAllString(normalized, "$1")
	}
	return normalized
}

func LinkCertainty(kind EntityKind, key string) Certainty {
	switch kind {
	case EntityCNPJ:
		return CertaintyExact
	case EntityContrato:
		if strings.IndexFunc(key, func(r rune) bool { return r >= 'A' && r <= 'Z' }) < 0 {
			return CertaintyWeak
		}
	}
	return CertaintyStrong
}

func ParseEntityInput(kind EntityKind, s string) (string, error) {
	switch kind {
	case EntityCNPJ:
		n, ok := NormalizeCNPJ(s)
		if !ok {
			return "", ErrInvalidCNPJ
		}
		return n, nil
	case EntityProcesso:
		n := nonDigitRe.ReplaceAllString(s, "")
		if len(n) < minProcessoDigits {
			return "", ErrInvalidInput
		}
		return n, nil
	case EntityContrato:
		n := strings.ToUpper(strings.ReplaceAll(strings.Join(strings.Fields(s), ""), "-", "/"))
		if !IsContratoShape(n) {
			return "", ErrInvalidInput
		}
		return EntityKey(kind, n), nil
	}
	return "", ErrInvalidInput
}

func ReportCertainty(kind EntityKind, weakest Certainty, sources int) Certainty {
	if kind != EntityCNPJ && sources > 1 {
		return CertaintyWeak
	}
	return weakest
}
