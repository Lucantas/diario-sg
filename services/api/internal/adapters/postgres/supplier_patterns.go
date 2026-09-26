package postgres

import (
	"context"
	"database/sql"
	"strings"

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
