package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type ReindexResult struct {
	Found     int `json:"found"`
	Reindexed int `json:"reindexed"`
	Failed    int `json:"failed"`
}

type ReindexGazettes struct {
	gazettes  ports.GazetteRepository
	storage   ports.FileStorage
	extractor ports.TextExtractor
	parsers   ports.ActParsers
	entities  ports.EntityExtractor
}

func NewReindexGazettes(g ports.GazetteRepository, s ports.FileStorage, e ports.TextExtractor, p ports.ActParsers, x ports.EntityExtractor) *ReindexGazettes {
	return &ReindexGazettes{gazettes: g, storage: s, extractor: e, parsers: p, entities: x}
}

func (uc *ReindexGazettes) Execute(ctx context.Context, from, to time.Time) (ReindexResult, error) {
	if to.Before(from) {
		return ReindexResult{}, fmt.Errorf("%w: período invertido", domain.ErrInvalidInput)
	}
	gazettes, err := uc.gazettes.ListByPeriod(ctx, from, to)
	if err != nil {
		return ReindexResult{}, fmt.Errorf("listar edições: %w", err)
	}
	res := ReindexResult{Found: len(gazettes)}
	var errs []error
	for _, g := range gazettes {
		if err := ctx.Err(); err != nil {
			return res, errors.Join(append(errs, err)...)
		}
		if err := uc.reindexOne(ctx, g); err != nil {
			res.Failed++
			errs = append(errs, fmt.Errorf("edição %s (%s): %w", g.PublishedAt.Format(time.DateOnly), g.StoragePath, err))
			continue
		}
		res.Reindexed++
	}
	return res, errors.Join(errs...)
}

func (uc *ReindexGazettes) reindexOne(ctx context.Context, g domain.Gazette) error {
	rc, err := uc.storage.Get(ctx, g.StoragePath)
	if err != nil {
		return fmt.Errorf("baixar: %w", err)
	}
	defer rc.Close()
	source := domain.SourceOrDefault(g.Source)
	text, err := uc.extractor.Extract(ctx, rc, source)
	if err != nil {
		return fmt.Errorf("extrair texto: %w", err)
	}
	parser := uc.parsers.For(source)
	number := parser.EditionNumber(text)
	if number == "" {
		number = g.EditionNumber
	}
	return uc.gazettes.ReplaceActs(ctx, g.ID, number, parseActs(parser, uc.entities, text))
}
