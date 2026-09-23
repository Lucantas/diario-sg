package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type searchSpy struct {
	ports.ActRepository
	got domain.ActFilter
}

func (s *searchSpy) Search(_ context.Context, f domain.ActFilter) ([]domain.ActHit, int, error) {
	s.got = f
	return []domain.ActHit{{}}, 1, nil
}

func TestActFeedUsesRecentOrderAndFeedLimit(t *testing.T) {
	spy := &searchSpy{}

	hits, err := NewActFeed(spy).Execute(context.Background(), domain.ActFilter{Query: "merenda", Limit: 20, Offset: 40})

	if err != nil || len(hits) != 1 {
		t.Fatalf("veio %v %v", hits, err)
	}
	if !spy.got.Recent || spy.got.Limit != domain.FeedLimit || spy.got.Offset != 0 {
		t.Fatalf("filtro inesperado: %+v", spy.got)
	}
}

func TestActFeedRejectsInvalidFilter(t *testing.T) {
	_, err := NewActFeed(&searchSpy{}).Execute(context.Background(), domain.ActFilter{Type: "bobagem"})

	if !errors.Is(err, domain.ErrInvalidFilter) {
		t.Fatalf("esperava ErrInvalidFilter, veio %v", err)
	}
}
