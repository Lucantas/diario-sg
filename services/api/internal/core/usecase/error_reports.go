package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type ErrorReports struct{ repo ports.ErrorReportRepository }

func NewErrorReports(r ports.ErrorReportRepository) *ErrorReports { return &ErrorReports{repo: r} }

func (uc *ErrorReports) Submit(ctx context.Context, gazetteID string, position int, title string, kind domain.ReportKind, message string) error {
	r, err := domain.NewErrorReport(gazetteID, position, title, kind, message)
	if err != nil {
		return err
	}
	return uc.repo.Create(ctx, &r)
}

func (uc *ErrorReports) List(ctx context.Context, status domain.ReportStatus) ([]domain.ErrorReport, error) {
	if !status.Valid() {
		return nil, domain.ErrInvalidReport
	}
	return uc.repo.List(ctx, status)
}

func (uc *ErrorReports) Close(ctx context.Context, id string, status domain.ReportStatus) error {
	if !status.ClosesReport() {
		return domain.ErrInvalidReport
	}
	return uc.repo.Close(ctx, id, status)
}
