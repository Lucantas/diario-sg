package postgres

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const licenseHeadRunes = 600

func (r *SupplierPatternRepo) LicenseActs(ctx context.Context) ([]domain.LicenseAct, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, e.key, g.published_at, a.title, left(a.body, $3)
		FROM acts a
		JOIN gazettes g ON g.id = a.gazette_id
		JOIN entity_links l ON l.record_kind = $1 AND l.source = g.source AND l.record_id = a.id::text
		JOIN entities e ON e.id = l.entity_id AND e.kind = 'cnpj'
		WHERE a.type = $2
		ORDER BY g.published_at, a.id`, domain.RecordAct, string(domain.ActLicencaAmbiental), licenseHeadRunes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LicenseAct
	for rows.Next() {
		var l domain.LicenseAct
		if err := rows.Scan(&l.ActID, &l.CNPJ, &l.PublishedAt, &l.Title, &l.Head); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
