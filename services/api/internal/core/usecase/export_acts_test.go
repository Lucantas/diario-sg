package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type exportSpy struct {
	ports.ActRepository
	got domain.ActFilter
}

func (s *exportSpy) Export(_ context.Context, f domain.ActFilter, _ func(domain.ActHit, int) error) error {
	s.got = f
	return nil
}

func TestExportActsIgnoresPaginationAndUsesTheExportLimit(t *testing.T) {
	spy := &exportSpy{}

	err := NewExportActs(spy).Execute(context.Background(),
		domain.ActFilter{Query: "a OU b", Limit: 20, Offset: 40}, func(domain.ActHit, int) error { return nil })

	if err != nil || spy.got.Limit != domain.ExportLimit || spy.got.Offset != 0 || spy.got.Query != "a or b" {
		t.Fatalf("filtro inesperado: %+v %v", spy.got, err)
	}
}

func TestExportActsRejectsInvalidFilter(t *testing.T) {
	err := NewExportActs(&exportSpy{}).Execute(context.Background(),
		domain.ActFilter{Organ: "TOTAL"}, func(domain.ActHit, int) error { return nil })

	if !errors.Is(err, domain.ErrInvalidFilter) {
		t.Fatalf("esperava ErrInvalidFilter, veio %v", err)
	}
}
