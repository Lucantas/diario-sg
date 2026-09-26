package postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type SupplierPatternRepo struct {
	*PanelRepo
	db *sql.DB
}

func NewSupplierPatternRepo(db *sql.DB) *SupplierPatternRepo {
	return &SupplierPatternRepo{PanelRepo: NewPanelRepo(db), db: db}
}

func (r *SupplierPatternRepo) SupplierProfiles(ctx context.Context) (map[string]domain.SupplierProfile, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT x.cnpj, coalesce(c.name, ''), x.headquarters, x.opened_at, coalesce(c.capital_cents, 0), coalesce(c.legal_nature, ''),
		       x.street, x.number, x.complement, x.district, x.city, x.uf, x.zip
		FROM rf_establishments x LEFT JOIN rf_companies c ON c.cnpj_base = x.cnpj_base`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]domain.SupplierProfile{}
	for rows.Next() {
		var p domain.SupplierProfile
		var street, number, complement, district, city, uf, zip string
		if err := rows.Scan(&p.CNPJ, &p.Name, &p.Headquarters, &p.OpenedAt, &p.CapitalCents, &p.LegalNature,
			&street, &number, &complement, &district, &city, &uf, &zip); err != nil {
			return nil, err
		}
		p.Address = joinNonEmpty(street, number, complement, district, city, uf, zip)
		out[p.CNPJ] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, r.addPartners(ctx, out)
}

func (r *SupplierPatternRepo) addPartners(ctx context.Context, profiles map[string]domain.SupplierProfile) error {
	rows, err := r.db.QueryContext(ctx, `SELECT cnpj_base, kind, name, document FROM rf_partners ORDER BY cnpj_base, name`)
	if err != nil {
		return err
	}
	defer rows.Close()
	byBase := map[string][]domain.PartnerKey{}
	for rows.Next() {
		var base string
		var k domain.PartnerKey
		if err := rows.Scan(&base, &k.Kind, &k.Name, &k.Document); err != nil {
			return err
		}
		byBase[base] = append(byBase[base], k)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for cnpj, p := range profiles {
		p.Partners = byBase[domain.CNPJBase(cnpj)]
		profiles[cnpj] = p
	}
	return nil
}

func (r *SupplierPatternRepo) AllSanctions(ctx context.Context) ([]domain.Sanction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT register, code, cnpj, name, category, starts_at, ends_at, published_at, process, organ, organ_uf,
		       sphere, scope, legal_basis, fine_cents, first_seen, last_seen
		FROM cgu_sanctions ORDER BY register, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSanctions(rows)
}

func joinNonEmpty(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, ", ")
}

func (r *SupplierPatternRepo) PaidCreditors(ctx context.Context) ([]domain.CreditorPaid, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT cnpj, array_agg(DISTINCT year ORDER BY year) FILTER (WHERE paid_cents > 0),
		       array_agg(DISTINCT unit ORDER BY unit), sum(paid_cents)
		FROM payments
		WHERE unit NOT ILIKE 'C_MARA%' AND function NOT IN ('ENCARGOS ESPECIAIS', 'PREVIDÊNCIA SOCIAL')
		GROUP BY cnpj HAVING sum(paid_cents) > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CreditorPaid
	for rows.Next() {
		var c domain.CreditorPaid
		var years []int64
		if err := rows.Scan(&c.CNPJ, pq.Array(&years), pq.Array(&c.Units), &c.PaidCents); err != nil {
			return nil, err
		}
		for _, y := range years {
			c.Years = append(c.Years, int(y))
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *SupplierPatternRepo) CitedInDiario(ctx context.Context) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key FROM entities WHERE kind = 'cnpj'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out[k] = true
	}
	return out, rows.Err()
}

func (r *SupplierPatternRepo) AllPNCPContracts(ctx context.Context) ([]domain.PNCPContract, error) {
	return NewPNCPRepo(r.db).AllPNCPContracts(ctx)
}

func (r *SupplierPatternRepo) CitedProcesses(ctx context.Context) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key FROM entities WHERE kind = 'processo'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out[k] = true
	}
	return out, rows.Err()
}

func (r *SupplierPatternRepo) LatestGazetteDay(ctx context.Context, source string) (time.Time, error) {
	var day sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT max(published_at) FROM gazettes WHERE source = $1`, source).Scan(&day)
	return day.Time, err
}

func (r *SupplierPatternRepo) PaymentsCoverage(ctx context.Context) (*domain.PaymentCoverage, error) {
	return NewPaymentRepo(r.db).PaymentsCoverage(ctx)
}

func (r *SupplierPatternRepo) PanelActsWithoutCNPJ(ctx context.Context, source string) ([]domain.PanelAct, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.type, a.title, a.organ, g.published_at, coalesce(a.main_value_cents, 0), a.declared_increase_bp, left(a.body, $4),
		       coalesce((SELECT array_agg(DISTINCT e.kind || ':' || e.key)
		                 FROM entity_links l JOIN entities e ON e.id = l.entity_id AND e.kind IN ('processo', 'contrato')
		                 WHERE l.source = $2 AND l.record_kind = $1 AND l.record_id = a.id::text), '{}')
		FROM acts a JOIN gazettes g ON g.id = a.gazette_id
		WHERE g.source = $2 AND a.type = ANY($3) AND (a.main_value_cents > 0 OR a.declared_increase_bp > 0)
		  AND NOT EXISTS (SELECT 1 FROM entity_links l JOIN entities e ON e.id = l.entity_id AND e.kind = 'cnpj'
		                  WHERE l.source = $2 AND l.record_kind = $1 AND l.record_id = a.id::text)`,
		domain.RecordAct, source, pq.Array(panelActTypes), domain.PanelHeadRunes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PanelAct
	for rows.Next() {
		var a domain.PanelAct
		var typ string
		if err := rows.Scan(&a.ActID, &typ, &a.Title, &a.Organ, &a.PublishedAt, &a.ValueCents, &a.DeclaredIncreaseBP, &a.Head, pq.Array(&a.Refs)); err != nil {
			return nil, err
		}
		a.Type = domain.ActType(typ)
		out = append(out, a)
	}
	return out, rows.Err()
}
