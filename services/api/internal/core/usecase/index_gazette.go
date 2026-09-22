// Package usecase contém os casos de uso da aplicação. Dependem apenas de
// domain e ports; nunca de adapters.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type IndexGazetteInput struct {
	EditionNumber string
	PublishedAt   time.Time
	SourceURL     string
	StoragePath   string
	Checksum      string
}

// IndexGazette extrai o texto de uma edição, separa em atos, grava e avisa.
type IndexGazette struct {
	gazettes  ports.GazetteRepository
	storage   ports.FileStorage
	extractor ports.TextExtractor
	parser    ports.ActParser
	entities  ports.EntityExtractor
	publisher ports.EventPublisher
	now       func() time.Time
}

func NewIndexGazette(g ports.GazetteRepository, s ports.FileStorage, e ports.TextExtractor, p ports.ActParser, x ports.EntityExtractor, pub ports.EventPublisher) *IndexGazette {
	return &IndexGazette{gazettes: g, storage: s, extractor: e, parser: p, entities: x, publisher: pub, now: time.Now}
}

func (uc *IndexGazette) Execute(ctx context.Context, in IndexGazetteInput) error {
	if in.StoragePath == "" || in.Checksum == "" || in.PublishedAt.IsZero() {
		return fmt.Errorf("%w: storage_path, checksum e published_at são obrigatórios", domain.ErrInvalidInput)
	}

	// Pub/Sub entrega "pelo menos uma vez". Se já indexamos, apenas
	// republicamos o evento (o passo seguinte também é idempotente).
	if id, found, err := uc.gazettes.FindIDByChecksum(ctx, in.Checksum); err != nil {
		return err
	} else if found {
		return uc.publisher.GazetteIndexed(ctx, id, -1)
	}

	rc, err := uc.storage.Get(ctx, in.StoragePath)
	if err != nil {
		return fmt.Errorf("baixar edição: %w", err)
	}
	defer rc.Close()

	text, err := uc.extractor.Extract(ctx, rc)
	if err != nil {
		return fmt.Errorf("extrair texto: %w", err)
	}

	acts := parseActs(uc.parser, uc.entities, text)
	number := in.EditionNumber
	if number == "" {
		number = uc.parser.EditionNumber(text)
	}
	g := &domain.Gazette{
		EditionNumber: number,
		PublishedAt:   in.PublishedAt,
		SourceURL:     in.SourceURL,
		StoragePath:   in.StoragePath,
		Checksum:      in.Checksum,
		IndexedAt:     uc.now(),
	}
	if err := uc.gazettes.SaveWithActs(ctx, g, acts); err != nil {
		return fmt.Errorf("gravar edição: %w", err)
	}
	return uc.publisher.GazetteIndexed(ctx, g.ID, len(acts))
}
