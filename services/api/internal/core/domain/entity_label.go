package domain

import (
	"regexp"
	"strings"
)

var labelNumberRe = regexp.MustCompile(`\d[\d./A-Za-z-]*[\dA-Za-z]|\d`)

func EntityLabel(kind EntityKind, value string) string {
	if kind == EntityCNPJ {
		return FormatCNPJ(value)
	}
	matches := labelNumberRe.FindAllString(strings.Join(strings.Fields(value), " "), -1)
	if len(matches) == 0 {
		return strings.TrimSpace(value)
	}
	return strings.ToUpper(matches[len(matches)-1])
}

func EntitySlug(label string) string { return strings.ReplaceAll(label, "/", "-") }

func FormatCNPJ(s string) string {
	n, ok := NormalizeCNPJ(s)
	if !ok {
		return s
	}
	return n[0:2] + "." + n[2:5] + "." + n[5:8] + "/" + n[8:12] + "-" + n[12:14]
}
