package postgres

import (
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func filterSQL(f domain.ActFilter, next int) (string, []any) {
	var b strings.Builder
	var args []any
	param := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(next+len(args)-1)
	}
	if f.Type != "" {
		b.WriteString(" AND a.type = " + param(string(f.Type)))
	}
	if f.Organ != "" {
		b.WriteString(" AND a.organ = ANY(" + param(pq.StringArray(domain.OrganAcronyms(f.Organ))) + ")")
	}
	if !f.From.IsZero() {
		b.WriteString(" AND g.published_at >= " + param(f.From.Format(time.DateOnly)) + "::date")
	}
	if !f.To.IsZero() {
		b.WriteString(" AND g.published_at <= " + param(f.To.Format(time.DateOnly)) + "::date")
	}
	if f.MinCents > 0 || f.MaxCents > 0 {
		b.WriteString(" AND EXISTS (SELECT 1 FROM act_entities v WHERE v.act_id = a.id AND v.kind = 'valor'")
		if f.MinCents > 0 {
			b.WriteString(" AND v.normalized::bigint >= " + param(f.MinCents))
		}
		if f.MaxCents > 0 {
			b.WriteString(" AND v.normalized::bigint <= " + param(f.MaxCents))
		}
		b.WriteString(")")
	}
	return b.String(), args
}
