package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type PoliticalAgentRepo struct{ db *sql.DB }

func NewPoliticalAgentRepo(db *sql.DB) *PoliticalAgentRepo { return &PoliticalAgentRepo{db: db} }

func (r *PoliticalAgentRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM political_agent_pay, councillors LIMIT 0`)
	return err
}

func (r *PoliticalAgentRepo) SavePoliticalAgents(ctx context.Context, load domain.PoliticalAgentLoad) error {
	if load.Body != domain.BodyPrefeitura && load.Body != domain.BodyCamara {
		return fmt.Errorf("%w: órgão %q", domain.ErrInvalidInput, load.Body)
	}
	for _, p := range load.Pay {
		if p.Body != load.Body {
			return fmt.Errorf("%s em %s: linha de %s numa carga de %s", p.Name, p.Month.Format("01/2006"), p.Body, load.Body)
		}
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM political_agent_pay WHERE body = $1 AND month BETWEEN $2 AND $3`, load.Body, load.From, load.To); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("political_agent_pay", "body", "month", "name", "name_key", "role", "office",
		"gross_cents", "discount_cents", "net_cents"))
	if err != nil {
		return err
	}
	defer closeQuietly(stmt)
	for _, p := range load.Pay {
		if _, err := stmt.ExecContext(ctx, p.Body, p.Month, p.Name, p.NameKey, p.Role, p.Office, p.GrossCents, p.DiscountCents, p.NetCents); err != nil {
			return fmt.Errorf("%s em %s: %w", p.Name, p.Month.Format("01/2006"), err)
		}
	}
	if err := closeCopy(ctx, stmt); err != nil {
		return err
	}
	for _, c := range load.Councillors {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO councillors (legislature, name, name_key, parliamentary_name, party, situation) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (legislature, name_key) DO UPDATE SET name = excluded.name, parliamentary_name = excluded.parliamentary_name,
				party = excluded.party, situation = excluded.situation`,
			c.Legislature, c.Name, c.NameKey, c.ParliamentaryName, c.Party, c.Situation); err != nil {
			return fmt.Errorf("vereador %s: %w", c.Name, err)
		}
	}
	return tx.Commit()
}

func (r *PoliticalAgentRepo) AgentPay(ctx context.Context) ([]domain.AgentPay, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT body, month, name, name_key, role, office, gross_cents, discount_cents, net_cents
		FROM political_agent_pay ORDER BY month, body, name_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AgentPay
	for rows.Next() {
		var p domain.AgentPay
		if err := rows.Scan(&p.Body, &p.Month, &p.Name, &p.NameKey, &p.Role, &p.Office, &p.GrossCents, &p.DiscountCents, &p.NetCents); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PoliticalAgentRepo) Councillors(ctx context.Context) ([]domain.Councillor, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT legislature, name, name_key, parliamentary_name, party, situation FROM councillors ORDER BY legislature, name_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Councillor
	for rows.Next() {
		var c domain.Councillor
		if err := rows.Scan(&c.Legislature, &c.Name, &c.NameKey, &c.ParliamentaryName, &c.Party, &c.Situation); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
