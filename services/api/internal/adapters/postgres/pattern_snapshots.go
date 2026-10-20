package postgres

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

var patternInputTables = []string{
	"gazettes", "acts", "act_entities", "entities", "entity_links", "payments", "pncp_contracts",
	"rf_companies", "rf_establishments", "rf_partners", "cgu_sanctions", "political_agent_pay", "partner_appointment_names",
}

type PatternSnapshotRepo struct{ db *sql.DB }

func NewPatternSnapshotRepo(db *sql.DB) *PatternSnapshotRepo { return &PatternSnapshotRepo{db: db} }

func (r *PatternSnapshotRepo) PatternInputsFingerprint(ctx context.Context) (string, error) {
	parts := []string{`'mes:' || to_char(now(), 'YYYY-MM')`}
	for _, table := range patternInputTables {
		parts = append(parts, fmt.Sprintf(`(SELECT '%[1]s:' || count(*) || ':' || coalesce(max(xmin::text::bigint), 0) FROM %[1]s)`, table))
	}
	var fingerprint string
	err := r.db.QueryRowContext(ctx, `SELECT concat_ws(',', `+strings.Join(parts, ", ")+`)`).Scan(&fingerprint)
	return fingerprint, err
}

type storedPatterns struct {
	Reports []domain.PatternReport
	Acts    map[string]domain.ActHit
}

func (r *PatternSnapshotRepo) LoadPatternSnapshot(ctx context.Context, fingerprint string) (domain.PatternSnapshot, bool, error) {
	var payload []byte
	err := r.db.QueryRowContext(ctx, `SELECT payload FROM pattern_snapshots WHERE fingerprint = $1`, fingerprint).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PatternSnapshot{}, false, nil
	}
	if err != nil {
		return domain.PatternSnapshot{}, false, err
	}
	zr, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return domain.PatternSnapshot{}, false, fmt.Errorf("padrões guardados ilegíveis: %w", err)
	}
	defer zr.Close()
	var stored storedPatterns
	if err := json.NewDecoder(zr).Decode(&stored); err != nil {
		return domain.PatternSnapshot{}, false, fmt.Errorf("padrões guardados ilegíveis: %w", err)
	}
	return domain.PatternSnapshot{Fingerprint: fingerprint, Reports: stored.Reports, Acts: stored.Acts}, true, nil
}

func (r *PatternSnapshotRepo) SavePatternSnapshot(ctx context.Context, snap domain.PatternSnapshot) error {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if err := json.NewEncoder(zw).Encode(storedPatterns{Reports: snap.Reports, Acts: snap.Acts}); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO pattern_snapshots (id, fingerprint, computed_at, payload) VALUES (1, $1, now(), $2)
		ON CONFLICT (id) DO UPDATE SET fingerprint = excluded.fingerprint, computed_at = excluded.computed_at, payload = excluded.payload`,
		snap.Fingerprint, buf.Bytes())
	return err
}
