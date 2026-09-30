package postgres

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

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

var contractingActTypes = []domain.ActType{domain.ActContrato, domain.ActAditivo, domain.ActDispensa, domain.ActLicitacao, domain.ActLicencaAmbiental}

func (r *SupplierPatternRepo) ContractingCNPJs(ctx context.Context) ([]string, error) {
	types := make([]string, len(contractingActTypes))
	for i, t := range contractingActTypes {
		types[i] = string(t)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT e.key
		FROM entities e
		JOIN entity_links l ON l.entity_id = e.id AND l.record_kind = $1
		JOIN acts a ON a.id::text = l.record_id
		WHERE e.kind = 'cnpj' AND a.type = ANY($2)
		ORDER BY e.key`, domain.RecordAct, pq.Array(types))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var cnpj string
		if err := rows.Scan(&cnpj); err != nil {
			return nil, err
		}
		out = append(out, cnpj)
	}
	return out, rows.Err()
}
