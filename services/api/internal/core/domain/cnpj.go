package domain

import "strings"

const cnpjDigits = 14

func NormalizeCNPJ(s string) (string, bool) {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '/' || r == '-' || r == ' ':
		default:
			return "", false
		}
	}
	if b.Len() != cnpjDigits {
		return "", false
	}
	return b.String(), true
}
