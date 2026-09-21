// Package postgres implementa os repositórios do core com database/sql.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

// Open abre um pool pequeno: o plano gratuito do Neon limita conexões e
// cada instância do Cloud Run tem seu próprio pool.
func Open(ctx context.Context, url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxIdleTime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// notFound traduz "sem linhas" e UUID malformado para domain.ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "22P02" { // invalid_text_representation
		return domain.ErrNotFound
	}
	return err
}

func nullableDate(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format("2006-01-02")
}
