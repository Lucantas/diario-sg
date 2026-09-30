package postgres

import (
	"context"
	"database/sql"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func RefreshPartnerAppointments(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY partner_appointment_names`)
	return err
}

func (r *SupplierPatternRepo) PartnerAppointments(ctx context.Context) ([]domain.PartnerAppointment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT name, act_id, published_at FROM partner_appointment_names ORDER BY name, published_at DESC, act_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PartnerAppointment
	for rows.Next() {
		var a domain.PartnerAppointment
		if err := rows.Scan(&a.Name, &a.ActID, &a.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SupplierPatternRepo) PoliticalAgentNames(ctx context.Context) ([]domain.PublicAgentName, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT ON (name_key) name, role, office
		FROM political_agent_pay ORDER BY name_key, month DESC, body`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PublicAgentName
	for rows.Next() {
		var a domain.PublicAgentName
		if err := rows.Scan(&a.Name, &a.Role, &a.Office); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
