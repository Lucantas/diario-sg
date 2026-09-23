package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type memReports struct {
	created []domain.ErrorReport
	closed  map[string]domain.ReportStatus
}

func (m *memReports) Create(_ context.Context, r *domain.ErrorReport) error {
	r.ID = "r1"
	m.created = append(m.created, *r)
	return nil
}
func (m *memReports) List(context.Context, domain.ReportStatus) ([]domain.ErrorReport, error) {
	return m.created, nil
}
func (m *memReports) Close(_ context.Context, id string, s domain.ReportStatus) error {
	if m.closed == nil {
		m.closed = map[string]domain.ReportStatus{}
	}
	m.closed[id] = s
	return nil
}

func TestSubmitErrorReportValidatesBeforeSaving(t *testing.T) {
	repo := &memReports{}
	uc := NewErrorReports(repo)

	err := uc.Submit(context.Background(), "g1", 2, "PORTARIA", domain.ReportWrongOrgan, "")
	bad := uc.Submit(context.Background(), "g1", 2, "PORTARIA", "bobagem", "")

	if err != nil || len(repo.created) != 1 || repo.created[0].Status != domain.ReportOpen {
		t.Fatalf("reporte válido: %v %+v", err, repo.created)
	}
	if !errors.Is(bad, domain.ErrInvalidReport) || len(repo.created) != 1 {
		t.Fatalf("reporte inválido não pode ser gravado: %v", bad)
	}
}

func TestCloseErrorReportNeedsAClosingStatus(t *testing.T) {
	repo := &memReports{}
	uc := NewErrorReports(repo)

	if err := uc.Close(context.Background(), "r1", domain.ReportOpen); !errors.Is(err, domain.ErrInvalidReport) {
		t.Fatalf("fechar como aberto deveria falhar, veio %v", err)
	}
	if err := uc.Close(context.Background(), "r1", domain.ReportResolved); err != nil || repo.closed["r1"] != domain.ReportResolved {
		t.Fatalf("fechar como resolvido: %v %v", err, repo.closed)
	}
	if _, err := uc.List(context.Background(), "bobagem"); !errors.Is(err, domain.ErrInvalidReport) {
		t.Fatalf("status desconhecido na listagem deveria falhar, veio %v", err)
	}
}
