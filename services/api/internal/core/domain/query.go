package domain

import (
	"regexp"
	"strings"
)

var orWord = regexp.MustCompile(`(^|\s)OU(\s|$)`)

func TranslateOperators(q string) string {
	parts := strings.Split(q, `"`)
	for i := 0; i < len(parts); i += 2 {
		for orWord.MatchString(parts[i]) {
			parts[i] = orWord.ReplaceAllString(parts[i], "${1}or${2}")
		}
	}
	return strings.Join(parts, `"`)
}
