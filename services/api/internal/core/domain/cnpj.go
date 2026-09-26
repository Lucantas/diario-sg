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

var (
	cnpjFirstWeights  = []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	cnpjSecondWeights = []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
)

func HasValidCheckDigits(digits string) bool {
	if len(digits) != cnpjDigits {
		return false
	}
	return cnpjCheckDigit(digits, cnpjFirstWeights) == int(digits[12]-'0') &&
		cnpjCheckDigit(digits, cnpjSecondWeights) == int(digits[13]-'0')
}

func cnpjCheckDigit(digits string, weights []int) int {
	sum := 0
	for i, w := range weights {
		sum += int(digits[i]-'0') * w
	}
	if rest := sum % 11; rest >= 2 {
		return 11 - rest
	}
	return 0
}
