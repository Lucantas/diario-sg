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
	if f.Source != "" {
		b.WriteString(" AND g.source = " + param(f.Source))
	}
	if f.Type != "" {
		b.WriteString(" AND a.type = " + param(string(f.Type)))
	}
	if f.Organ != "" {
		b.WriteString(" AND a.organ = ANY(" + param(pq.StringArray(domain.OrganAcronyms(f.Organ))) + ")")
	}
	if theme, err := domain.ParseTheme(f.Theme); err == nil && !theme.IsZero() {
		b.WriteString(themeSQL(theme, param))
	}
	if !f.From.IsZero() {
		b.WriteString(" AND g.published_at >= " + param(f.From.Format(time.DateOnly)) + "::date")
	}
	if !f.To.IsZero() {
		b.WriteString(" AND g.published_at <= " + param(f.To.Format(time.DateOnly)) + "::date")
	}
	if f.Modality != "" {
		b.WriteString(" AND a.modality = " + param(string(f.Modality)))
	}
	if f.MainMinCents > 0 {
		b.WriteString(" AND a.main_value_cents >= " + param(f.MainMinCents))
	}
	if f.MainMaxCents > 0 {
		b.WriteString(" AND a.main_value_cents <= " + param(f.MainMaxCents))
	}
	if f.Name != "" && f.Query != "" {
		b.WriteString(" AND a.search @@ websearch_to_tsquery('" + tsConfig + "', " + param(f.NamePhrase()) + ")")
	}
	if f.ExcludesNameLists() {
		b.WriteString(" AND a.name_lines < " + param(domain.NameListMinLines))
	}
	if f.Entity != nil {
		b.WriteString(" AND a.id IN (SELECT l.record_id::uuid FROM entities e JOIN entity_links l ON l.entity_id = e.id AND l.record_kind = '" +
			domain.RecordAct + "' WHERE e.kind = " + param(string(f.Entity.Kind)) + " AND e.key = " + param(f.Entity.Key) + ")")
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

func themeSQL(t domain.Theme, param func(any) string) string {
	var organs []string
	for _, o := range t.Organs {
		organs = append(organs, domain.OrganAcronyms(o)...)
	}
	return " AND (a.organ = ANY(" + param(pq.StringArray(organs)) + ")" +
		" OR (a.organ = " + param(t.PartialOrgan) + " AND regexp_replace(lower(unaccent(a.body)), " + param(t.PartialOrganNameSQLRegex()) +
		", '', 'g') !~ " + param(t.PartialOrganExcludeSQLRegex()) + ")" +
		" OR (lower(unaccent(a.title)) ~ " + param(t.TermsSQLRegex()) + " AND lower(unaccent(a.title)) !~ " + param(t.ExcludeSQLRegex()) + "))"
}
