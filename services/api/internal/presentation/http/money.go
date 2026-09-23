package http

import (
	"regexp"
	"strconv"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

var reaisRe = regexp.MustCompile(`^(\d{1,13})(?:\.(\d{1,2}))?$`)

func parseReais(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	m := reaisRe.FindStringSubmatch(s)
	if m == nil {
		return 0, domain.ErrInvalidFilter
	}
	whole, _ := strconv.ParseInt(m[1], 10, 64)
	cents, _ := strconv.ParseInt((m[2] + "00")[:2], 10, 64)
	return whole*100 + cents, nil
}
