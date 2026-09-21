package domain

import "strings"

const cnpjDigits = 14

// NormalizeCNPJ aceita "12.345.678/0001-90" ou "12345678000190" e devolve só
// os 14 dígitos. Não valida dígitos verificadores: o Diário publica CNPJs
// com erro de digitação e o jornalista precisa encontrá-los mesmo assim.
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
