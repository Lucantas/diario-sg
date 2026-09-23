package postgres

import (
	"context"
	"database/sql"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func linkActs(ctx context.Context, tx *sql.Tx, gazetteID string) error {
	_, err := tx.ExecContext(ctx, `SELECT link_diario_acts($1)`, gazetteID)
	return err
}

func unlinkActs(ctx context.Context, tx *sql.Tx, gazetteID string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM entity_links
		WHERE source = $2 AND record_kind = $3
		  AND record_id IN (SELECT id::text FROM acts WHERE gazette_id = $1)`,
		gazetteID, domain.SourceDiarioPrefeitura, domain.RecordAct)
	return err
}
