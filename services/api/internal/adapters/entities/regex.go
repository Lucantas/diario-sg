package entities

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type Regex struct{}

func New() Regex { return Regex{} }

var (
	cnpjRe = regexp.MustCompile(`\b\d{2}\.?\d{3}\.?\d{3}/\d{4}-\s?\d{2}\b`)

	valorRe    = regexp.MustCompile(`R\$\s?(\d{1,3}(?:\.\d{3})+|\d+),(\d{2})\b`)
	contratoRe = regexp.MustCompile(`(?i)\bcontrato(?:\s+de\s+[a-zçãõáéíóúâêô]+)?(?:\s+[A-Z]{2,10})?\s*(?:n\s*[º°.o]*\s*[:.]?\s*)?(\d{1,5}(?:/[A-Za-z0-9]{1,12}){1,3})`)
	processoRe = regexp.MustCompile(`(?i)\b(?:processo|procedimento)(?:\s+administrativo|\s+sei!?)?\s*(?:n\s*[º°.o]*|no)?\s*[:.]?\s*(\d{1,3}\.\d{3,6}/\d{4}-\d|\d{1,6}\.?\d{0,3}/\d{4})`)
	digitsRe   = regexp.MustCompile(`\D`)

	contratoShapeRe = regexp.MustCompile(`^\d{1,5}(?:/[A-Z]{2,10})?/\d{4}(?:/[A-Z]{2,10})?$`)
)

func (Regex) Extract(body string) []domain.Entity {
	var out []domain.Entity
	seen := map[string]bool{}
	add := func(kind domain.EntityKind, value, normalized string) {
		key := string(kind) + ":" + normalized
		if normalized == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, domain.Entity{Kind: kind, Value: strings.TrimSpace(value), Normalized: normalized})
	}

	for _, m := range cnpjRe.FindAllString(body, -1) {
		if n, ok := domain.NormalizeCNPJ(m); ok {
			add(domain.EntityCNPJ, m, n)
		}
	}
	for _, m := range valorRe.FindAllStringSubmatch(body, -1) {
		reais := digitsRe.ReplaceAllString(m[1], "")
		cents, err := strconv.ParseInt(reais+m[2], 10, 64)
		if err != nil {
			continue
		}
		add(domain.EntityValor, m[0], strconv.FormatInt(cents, 10))
	}
	for _, m := range contratoRe.FindAllStringSubmatch(body, -1) {
		n := normalizeNumber(m[1])
		if contratoShapeRe.MatchString(n) {
			add(domain.EntityContrato, m[0], n)
		}
	}
	for _, m := range processoRe.FindAllStringSubmatch(body, -1) {
		add(domain.EntityProcesso, m[0], digitsRe.ReplaceAllString(m[1], ""))
	}
	return out
}

func normalizeNumber(s string) string {
	s = strings.ToUpper(strings.Join(strings.Fields(s), ""))
	return strings.TrimRight(s, ".")
}
