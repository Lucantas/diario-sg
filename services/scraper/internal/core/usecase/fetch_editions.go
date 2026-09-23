package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/ports"
)

const MaxFileSize = 100 << 20

type FetchEditions struct {
	source    ports.EditionSource
	storage   ports.ObjectStorage
	publisher ports.EventPublisher
	now       func() time.Time
}

func NewFetchEditions(src ports.EditionSource, st ports.ObjectStorage, pub ports.EventPublisher) *FetchEditions {
	return &FetchEditions{source: src, storage: st, publisher: pub, now: time.Now}
}

type FetchResult struct {
	Found   int `json:"found"`
	Stored  int `json:"stored"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

func (uc *FetchEditions) Execute(ctx context.Context, lookback time.Duration) (FetchResult, error) {
	to := uc.now()
	return uc.ExecuteRange(ctx, to.Add(-lookback), to)
}

func (uc *FetchEditions) ExecuteRange(ctx context.Context, from, to time.Time) (FetchResult, error) {
	if to.Before(from) {
		return FetchResult{}, fmt.Errorf("período inválido: %s depois de %s", from.Format(time.DateOnly), to.Format(time.DateOnly))
	}
	editions, err := uc.source.ListEditions(ctx, from, to)
	if err != nil {
		return FetchResult{}, fmt.Errorf("listar edições: %w", err)
	}

	res := FetchResult{Found: len(editions)}
	var errs []error
	for _, e := range editions {
		done, err := uc.storage.Exists(ctx, e.MarkerPath())
		if err != nil {
			res.Failed++
			errs = append(errs, err)
			continue
		}
		if done {
			res.Skipped++
			continue
		}
		if err := uc.fetchOne(ctx, e); err != nil {
			res.Failed++
			errs = append(errs, fmt.Errorf("edição %s: %w", e.StoragePath(), err))
			continue
		}
		res.Stored++
	}
	return res, errors.Join(errs...)
}

func (uc *FetchEditions) fetchOne(ctx context.Context, e domain.Edition) error {
	rc, err := uc.source.Download(ctx, e)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(io.LimitReader(rc, MaxFileSize+1))
	if err != nil {
		return fmt.Errorf("leitura: %w", err)
	}
	if len(data) > MaxFileSize {
		return fmt.Errorf("arquivo maior que %d bytes", MaxFileSize)
	}
	sum := sha256.Sum256(data)

	path := e.StoragePath()
	if err := uc.storage.Put(ctx, path, "application/pdf", bytes.NewReader(data)); err != nil {
		return fmt.Errorf("armazenar: %w", err)
	}
	fetched := domain.FetchedEdition{
		Edition:        e,
		StoragePath:    path,
		ChecksumSHA256: hex.EncodeToString(sum[:]),
		FetchedAt:      uc.now(),
	}
	if err := uc.publisher.EditionFetched(ctx, fetched); err != nil {
		return fmt.Errorf("publicar evento: %w", err)
	}
	if err := uc.storage.Put(ctx, e.MarkerPath(), "text/plain", bytes.NewReader([]byte(fetched.ChecksumSHA256))); err != nil {
		return fmt.Errorf("gravar marcador: %w", err)
	}
	return nil
}
