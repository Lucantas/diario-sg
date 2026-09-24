package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type GroupActs struct{ groups ports.ActGrouper }

func NewGroupActs(g ports.ActGrouper) *GroupActs { return &GroupActs{groups: g} }

func (uc *GroupActs) Execute(ctx context.Context, q domain.GroupQuery) (domain.ActGroups, error) {
	if err := q.Normalize(); err != nil {
		return domain.ActGroups{}, err
	}
	return uc.groups.Group(ctx, q)
}
