package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const registryLinkRole = "cadastro"

type RegistryRepo struct{ db *sql.DB }

func NewRegistryRepo(db *sql.DB) *RegistryRepo { return &RegistryRepo{db: db} }

func (r *RegistryRepo) Ready(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `SELECT 1 FROM rf_companies, rf_establishments, rf_partners LIMIT 0`)
	return err
}

func (r *RegistryRepo) CitedCNPJs(ctx context.Context) ([]string, error) {
	return citedCNPJs(ctx, r.db)
}

func citedCNPJs(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT key FROM entities WHERE kind = 'cnpj' AND length(key) = 14
		UNION
		SELECT DISTINCT cnpj FROM payments
		ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *RegistryRepo) Replace(ctx context.Context, month time.Time, load domain.RegistryLoad) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range []string{`DELETE FROM rf_partners`, `DELETE FROM rf_establishments`, `DELETE FROM rf_companies`} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM entity_links WHERE source = $1`, domain.SourceReceita); err != nil {
		return err
	}
	for _, step := range []func(context.Context, *sql.Tx, time.Time, domain.RegistryLoad) error{insertCompanies, insertEstablishments, insertPartners, linkEstablishments} {
		if err := step(ctx, tx, month, load); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func insertCompanies(ctx context.Context, tx *sql.Tx, month time.Time, load domain.RegistryLoad) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("rf_companies", "cnpj_base", "name", "legal_nature", "capital_cents", "size", "reference_month"))
	if err != nil {
		return err
	}
	for _, c := range load.Companies {
		if _, err := stmt.ExecContext(ctx, c.Base, c.Name, c.LegalNature, c.CapitalCents, c.Size, month); err != nil {
			return fmt.Errorf("empresa %s: %w", c.Base, err)
		}
	}
	return closeCopy(ctx, stmt)
}

func insertEstablishments(ctx context.Context, tx *sql.Tx, month time.Time, load domain.RegistryLoad) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("rf_establishments", "cnpj", "cnpj_base", "headquarters", "trade_name", "status",
		"status_since", "status_reason", "opened_at", "main_activity_code", "main_activity", "other_activities",
		"street", "number", "complement", "district", "zip", "city", "uf", "reference_month"))
	if err != nil {
		return err
	}
	for _, e := range load.Establishments {
		others, err := json.Marshal(activitiesJSON(e.OtherActivities))
		if err != nil {
			return err
		}
		if _, err := stmt.ExecContext(ctx, e.CNPJ, domain.CNPJBase(e.CNPJ), e.Headquarters, e.TradeName, e.Status,
			e.StatusSince, e.StatusReason, e.OpenedAt, e.MainActivity.Code, e.MainActivity.Description, string(others),
			e.Street, e.Number, e.Complement, e.District, e.ZIP, e.City, e.UF, month); err != nil {
			return fmt.Errorf("estabelecimento %s: %w", e.CNPJ, err)
		}
	}
	return closeCopy(ctx, stmt)
}

func insertPartners(ctx context.Context, tx *sql.Tx, month time.Time, load domain.RegistryLoad) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("rf_partners", "cnpj_base", "kind", "name", "document", "role", "since", "reference_month"))
	if err != nil {
		return err
	}
	for _, p := range load.Partners {
		if _, err := stmt.ExecContext(ctx, p.Base, int(p.Kind), p.Name, p.Document, p.Role, p.Since, month); err != nil {
			return fmt.Errorf("sócio de %s: %w", p.Base, err)
		}
	}
	return closeCopy(ctx, stmt)
}

func closeCopy(ctx context.Context, stmt *sql.Stmt) error {
	if _, err := stmt.ExecContext(ctx); err != nil {
		return err
	}
	return stmt.Close()
}

func linkEstablishments(ctx context.Context, tx *sql.Tx, _ time.Time, _ domain.RegistryLoad) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO entity_links (entity_id, source, record_kind, record_id, role, certainty, evidence)
		SELECT e.id, $1, $2, x.cnpj, $3, $4, coalesce(c.name, '')
		FROM rf_establishments x
		JOIN entities e ON e.kind = 'cnpj' AND e.key = x.cnpj
		LEFT JOIN rf_companies c ON c.cnpj_base = x.cnpj_base
		ON CONFLICT DO NOTHING`,
		domain.SourceReceita, domain.RecordEstablishment, registryLinkRole, domain.CertaintyExact)
	return err
}

type activityJSON struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

func activitiesJSON(as []domain.Activity) []activityJSON {
	out := make([]activityJSON, len(as))
	for i, a := range as {
		out[i] = activityJSON{Code: a.Code, Description: a.Description}
	}
	return out
}

func (r *RegistryRepo) RegistryMonth(ctx context.Context) (*time.Time, error) {
	var month sql.NullTime
	if err := r.db.QueryRowContext(ctx, `SELECT max(reference_month) FROM rf_establishments`).Scan(&month); err != nil {
		return nil, err
	}
	if !month.Valid {
		return nil, nil
	}
	return &month.Time, nil
}

func (r *RegistryRepo) RegistryByCNPJ(ctx context.Context, cnpj string) (*domain.CompanyRegistry, error) {
	reg := domain.CompanyRegistry{}
	e := &reg.Establishment
	var others []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT x.cnpj, x.headquarters, x.trade_name, x.status, x.status_since, x.status_reason, x.opened_at,
		       x.main_activity_code, x.main_activity, x.other_activities, x.street, x.number, x.complement,
		       x.district, x.zip, x.city, x.uf, x.reference_month,
		       coalesce(c.cnpj_base, x.cnpj_base), coalesce(c.name, ''), coalesce(c.legal_nature, ''),
		       coalesce(c.capital_cents, 0), coalesce(c.size, '')
		FROM rf_establishments x LEFT JOIN rf_companies c ON c.cnpj_base = x.cnpj_base
		WHERE x.cnpj = $1`, cnpj).Scan(
		&e.CNPJ, &e.Headquarters, &e.TradeName, &e.Status, &e.StatusSince, &e.StatusReason, &e.OpenedAt,
		&e.MainActivity.Code, &e.MainActivity.Description, &others, &e.Street, &e.Number, &e.Complement,
		&e.District, &e.ZIP, &e.City, &e.UF, &reg.Month,
		&reg.Company.Base, &reg.Company.Name, &reg.Company.LegalNature, &reg.Company.CapitalCents, &reg.Company.Size)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var parsed []activityJSON
	if err := json.Unmarshal(others, &parsed); err != nil {
		return nil, err
	}
	e.OtherActivities = make([]domain.Activity, len(parsed))
	for i, a := range parsed {
		e.OtherActivities[i] = domain.Activity{Code: a.Code, Description: a.Description}
	}
	reg.Partners, err = r.partners(ctx, reg.Company.Base)
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *RegistryRepo) partners(ctx context.Context, base string) ([]domain.RegistryPartner, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT kind, name, document, role, since FROM rf_partners WHERE cnpj_base = $1 ORDER BY since NULLS LAST, name`, base)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.RegistryPartner{}
	for rows.Next() {
		p := domain.RegistryPartner{Base: base}
		var kind int
		if err := rows.Scan(&kind, &p.Name, &p.Document, &p.Role, &p.Since); err != nil {
			return nil, err
		}
		p.Kind = domain.PartnerKind(kind)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *RegistryRepo) NamesByCNPJ(ctx context.Context, cnpjs []string) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT x.cnpj, c.name FROM rf_establishments x JOIN rf_companies c ON c.cnpj_base = x.cnpj_base
		WHERE x.cnpj = ANY($1)`, pq.Array(cnpjs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var cnpj, name string
		if err := rows.Scan(&cnpj, &name); err != nil {
			return nil, err
		}
		out[cnpj] = name
	}
	return out, rows.Err()
}
