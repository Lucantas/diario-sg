package domain

import (
	"errors"
	"slices"
	"testing"
)

func TestGroupQueryNormalize(t *testing.T) {
	q := GroupQuery{By: GroupByCNPJ, Limit: 500}
	if err := q.Normalize(); err != nil || q.Limit != MaxGroupLimit {
		t.Fatalf("limite deveria ir a %d: %d %v", MaxGroupLimit, q.Limit, err)
	}
	q = GroupQuery{By: GroupByOrgan}
	if err := q.Normalize(); err != nil || q.Limit != DefaultGroupLimit {
		t.Fatalf("limite padrão: %d %v", q.Limit, err)
	}
	for _, bad := range []GroupQuery{{By: "fornecedor"}, {By: GroupByCNPJ, Filter: ActFilter{Type: "xyz"}}} {
		if err := bad.Normalize(); !errors.Is(err, ErrInvalidFilter) {
			t.Errorf("%+v deveria ser filtro inválido: %v", bad, err)
		}
	}
}

func TestGroupQueryExcludesPublicBodiesOnlyForCNPJ(t *testing.T) {
	byCNPJ := GroupQuery{By: GroupByCNPJ}.ExcludedKeys()
	if !slices.Contains(byCNPJ, "28636579") || !slices.Contains(byCNPJ, "32538167000105") {
		t.Fatalf("agrupar por CNPJ deixa de fora o Município e o SG-PREVI: %v", byCNPJ)
	}
	if got := (GroupQuery{By: GroupByCNPJ, IncludePublic: true}).ExcludedKeys(); len(got) != 0 {
		t.Fatalf("incluir órgãos públicos não exclui nada: %v", got)
	}
	if got := (GroupQuery{By: GroupByProcesso}).ExcludedKeys(); len(got) != 0 {
		t.Fatalf("processo não tem órgão público a excluir: %v", got)
	}
}
