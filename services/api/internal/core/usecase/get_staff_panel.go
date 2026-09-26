package usecase

import (
	"context"
	"fmt"
	"slices"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type GetStaffPanel struct {
	staff ports.StaffReader
	acts  ports.ActMonthCounter
}

func NewGetStaffPanel(staff ports.StaffReader, acts ports.ActMonthCounter) *GetStaffPanel {
	return &GetStaffPanel{staff: staff, acts: acts}
}

func (uc *GetStaffPanel) Execute(ctx context.Context, unit string) (domain.StaffPanel, error) {
	units, err := uc.staff.StaffUnits(ctx)
	if err != nil {
		return domain.StaffPanel{}, err
	}
	if unit != "" && !slices.Contains(units, unit) {
		return domain.StaffPanel{}, fmt.Errorf("%w: unidade %q", domain.ErrInvalidInput, unit)
	}
	rows, err := uc.staff.StaffRows(ctx, unit)
	if err != nil {
		return domain.StaffPanel{}, err
	}
	source := domain.StaffDiarioSource(unit)
	counts, err := uc.acts.MonthlyActCounts(ctx, []domain.ActType{domain.ActNomeacao, domain.ActExoneracao}, source)
	if err != nil {
		return domain.StaffPanel{}, err
	}
	return domain.StaffPanel{Units: units, Unit: unit, DiarioSource: source, Groups: domain.StaffGroupsOf(rows),
		Months: domain.BuildStaffPanel(rows, counts)}, nil
}
