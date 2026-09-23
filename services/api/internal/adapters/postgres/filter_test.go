package postgres

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestFilterSQLEmptyFilterAddsNothing(t *testing.T) {
	where, args := filterSQL(domain.ActFilter{}, 5)

	if where != "" || len(args) != 0 {
		t.Errorf("filtro vazio deveria ser vazio: %q %v", where, args)
	}
}

func TestFilterSQLNumbersParametersFromNext(t *testing.T) {
	f := domain.ActFilter{Type: domain.ActContrato, Organ: "SEMED",
		From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		MinCents: 100, MaxCents: 900}

	where, args := filterSQL(f, 5)

	want := " AND a.type = $5 AND a.organ = $6 AND g.published_at >= $7::date AND g.published_at <= $8::date" +
		" AND EXISTS (SELECT 1 FROM act_entities v WHERE v.act_id = a.id AND v.kind = 'valor'" +
		" AND v.normalized::bigint >= $9 AND v.normalized::bigint <= $10)"
	if where != want {
		t.Errorf("veio\n%s\nesperava\n%s", where, want)
	}
	if !reflect.DeepEqual(args, []any{"contrato", "SEMED", "2024-01-01", "2024-12-31", int64(100), int64(900)}) {
		t.Errorf("argumentos inesperados: %v", args)
	}
}

func TestFilterSQLOnlyMaximum(t *testing.T) {
	where, args := filterSQL(domain.ActFilter{MaxCents: 900}, 3)

	want := " AND EXISTS (SELECT 1 FROM act_entities v WHERE v.act_id = a.id AND v.kind = 'valor' AND v.normalized::bigint <= $3)"
	if where != want || !reflect.DeepEqual(args, []any{int64(900)}) {
		t.Errorf("veio %q %v", where, args)
	}
}

func TestOrderSQLByRelevanceOrByDate(t *testing.T) {
	recent := orderSQL(domain.ActFilter{Recent: true})
	if !strings.Contains(recent, "g.published_at DESC") || strings.Contains(recent, "ts_rank") {
		t.Errorf("ordem cronológica inesperada: %s", recent)
	}
	if relevance := orderSQL(domain.ActFilter{}); !strings.Contains(relevance, "ts_rank") {
		t.Errorf("ordem por relevância inesperada: %s", relevance)
	}
}
