package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type recordingGrouper struct{ got domain.GroupQuery }

func (r *recordingGrouper) Group(_ context.Context, q domain.GroupQuery) (domain.ActGroups, error) {
	r.got = q
	return domain.ActGroups{MatchedActs: 1}, nil
}

func TestGroupActsNormalizesBeforeQuerying(t *testing.T) {
	g := &recordingGrouper{}

	res, err := NewGroupActs(g).Execute(context.Background(), domain.GroupQuery{By: domain.GroupByCNPJ, Limit: 999})

	if err != nil || res.MatchedActs != 1 || g.got.Limit != domain.MaxGroupLimit {
		t.Fatalf("consulta normalizada: %+v %v", g.got, err)
	}
}

func TestGroupActsRejectsUnknownGrouping(t *testing.T) {
	g := &recordingGrouper{}

	_, err := NewGroupActs(g).Execute(context.Background(), domain.GroupQuery{By: "empresa"})

	if !errors.Is(err, domain.ErrInvalidFilter) || g.got.By != "" {
		t.Fatalf("agrupamento desconhecido não chega ao banco: %v %+v", err, g.got)
	}
}
