package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeStaffReader struct{ rows []domain.StaffRow }

func (f fakeStaffReader) StaffRows(_ context.Context, unit string) ([]domain.StaffRow, error) {
	var out []domain.StaffRow
	for _, r := range f.rows {
		if unit == "" || r.Unit == unit {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f fakeStaffReader) StaffUnits(context.Context) ([]string, error) {
	return []string{"CÂMARA SÃO GONÇALO", "PREFEITURA SÃO GONÇALO"}, nil
}

type recActCounter struct{ source string }

func (r *recActCounter) MonthlyActCounts(_ context.Context, _ []domain.ActType, source string) ([]domain.MonthlyActCount, error) {
	r.source = source
	return nil, nil
}

func TestGetStaffPanelFiltersTheUnitAndItsDiario(t *testing.T) {
	jan := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	reader := fakeStaffReader{rows: []domain.StaffRow{
		{Month: jan, Unit: "CÂMARA SÃO GONÇALO", Group: "Comissionado", Headcount: 180},
		{Month: jan, Unit: "PREFEITURA SÃO GONÇALO", Group: "Efetivo", Headcount: 7000},
	}}
	counter := &recActCounter{}

	panel, err := NewGetStaffPanel(reader, counter).Execute(context.Background(), "CÂMARA SÃO GONÇALO")

	if err != nil || len(panel.Units) != 2 || len(panel.Months) != 1 || panel.Months[0].Headcount != 180 || counter.source != domain.SourceDiarioCamara {
		t.Fatalf("veio %+v %v (fonte %s)", panel, err, counter.source)
	}
}

func TestGetStaffPanelRejectsAnUnknownUnit(t *testing.T) {
	_, err := NewGetStaffPanel(fakeStaffReader{}, &recActCounter{}).Execute(context.Background(), "OUTRA")

	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("esperava entrada inválida, veio %v", err)
	}
}
