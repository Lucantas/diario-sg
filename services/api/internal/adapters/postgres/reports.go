package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const pgForeignKeyViolation = "23503"

type ErrorReportRepo struct{ db *sql.DB }

func NewErrorReportRepo(db *sql.DB) *ErrorReportRepo { return &ErrorReportRepo{db: db} }

func (r *ErrorReportRepo) Create(ctx context.Context, rep *domain.ErrorReport) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO error_reports (gazette_id, position, act_title, kind, message, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		rep.GazetteID, rep.Position, rep.ActTitle, string(rep.Kind), rep.Message, string(rep.Status),
	).Scan(&rep.ID, &rep.CreatedAt)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == pgForeignKeyViolation {
		return domain.ErrNotFound
	}
	return notFound(err)
}

func (r *ErrorReportRepo) List(ctx context.Context, status domain.ReportStatus) ([]domain.ErrorReport, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT e.id, e.gazette_id, e.position, e.act_title, e.kind, e.message, e.status, e.created_at,
		       coalesce(e.closed_at, 'epoch'), g.edition_number, g.published_at, coalesce(a.page_start, 0)
		FROM error_reports e
		JOIN gazettes g ON g.id = e.gazette_id
		LEFT JOIN acts a ON a.gazette_id = e.gazette_id AND a.position = e.position
		WHERE e.status = $1
		ORDER BY e.created_at`, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ErrorReport
	for rows.Next() {
		var rep domain.ErrorReport
		var kind, st string
		if err := rows.Scan(&rep.ID, &rep.GazetteID, &rep.Position, &rep.ActTitle, &kind, &rep.Message, &st, &rep.CreatedAt,
			&rep.ClosedAt, &rep.EditionNumber, &rep.PublishedAt, &rep.PageStart); err != nil {
			return nil, err
		}
		rep.Kind, rep.Status = domain.ReportKind(kind), domain.ReportStatus(st)
		out = append(out, rep)
	}
	return out, rows.Err()
}

func (r *ErrorReportRepo) Close(ctx context.Context, id string, status domain.ReportStatus) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE error_reports SET status = $2, closed_at = now() WHERE id = $1 AND status = 'aberto'`, id, string(status))
	if err != nil {
		return notFound(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
