package postgres

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type PanelRepo struct{ db *sql.DB }

func NewPanelRepo(db *sql.DB) *PanelRepo { return &PanelRepo{db: db} }

var panelActTypes = []string{string(domain.ActContrato), string(domain.ActDispensa), string(domain.ActLicitacao), string(domain.ActAta)}

func (r *PanelRepo) PanelActs(ctx context.Context, source string) ([]domain.PanelAct, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.type, a.title, a.organ, g.published_at, a.main_value_cents, left(a.body, 400),
		       coalesce((SELECT array_agg(DISTINCT e.key)
		                 FROM entity_links l JOIN entities e ON e.id = l.entity_id AND e.kind = 'cnpj'
		                 WHERE l.source = $2 AND l.record_kind = $1 AND l.record_id = a.id::text), '{}'),
		       coalesce((SELECT array_agg(DISTINCT e.kind || ':' || e.key)
		                 FROM entity_links l JOIN entities e ON e.id = l.entity_id AND e.kind IN ('processo', 'contrato')
		                 WHERE l.source = $2 AND l.record_kind = $1 AND l.record_id = a.id::text), '{}')
		FROM acts a JOIN gazettes g ON g.id = a.gazette_id
		WHERE g.source = $2 AND a.type = ANY($3) AND a.main_value_cents > 0`,
		domain.RecordAct, source, pq.Array(panelActTypes))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PanelAct
	for rows.Next() {
		var a domain.PanelAct
		var typ string
		var cnpjs []string
		if err := rows.Scan(&a.ActID, &typ, &a.Title, &a.Organ, &a.PublishedAt, &a.ValueCents, &a.Head, pq.Array(&cnpjs), pq.Array(&a.Refs)); err != nil {
			return nil, err
		}
		a.Type = domain.ActType(typ)
		out = append(out, actPerSupplier(a, cnpjs)...)
	}
	return out, rows.Err()
}

func actPerSupplier(a domain.PanelAct, cnpjs []string) []domain.PanelAct {
	out := make([]domain.PanelAct, 0, len(cnpjs))
	for i, cnpj := range cnpjs {
		per := a
		per.CNPJ = cnpj
		per.OtherCNPJs = append(append([]string{}, cnpjs[:i]...), cnpjs[i+1:]...)
		out = append(out, per)
	}
	return out
}

func (r *PanelRepo) HitsByIDs(ctx context.Context, ids []string) ([]domain.ActHit, error) {
	return hitsByIDs(ctx, r.db, ids)
}
