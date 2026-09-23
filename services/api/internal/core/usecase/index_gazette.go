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
	Source        string
}

type IndexGazette struct {
	gazettes  ports.GazetteRepository
	storage   ports.FileStorage
	extractor ports.TextExtractor
	parsers   ports.ActParsers
	entities  ports.EntityExtractor
	publisher ports.EventPublisher
	now       func() time.Time
}

func NewIndexGazette(g ports.GazetteRepository, s ports.FileStorage, e ports.TextExtractor, p ports.ActParsers, x ports.EntityExtractor, pub ports.EventPublisher) *IndexGazette {
	return &IndexGazette{gazettes: g, storage: s, extractor: e, parsers: p, entities: x, publisher: pub, now: time.Now}
}

func (uc *IndexGazette) Execute(ctx context.Context, in IndexGazetteInput) error {
	if in.StoragePath == "" || in.Checksum == "" || in.PublishedAt.IsZero() {
		return fmt.Errorf("%w: storage_path, checksum e published_at são obrigatórios", domain.ErrInvalidInput)
	}
	source := domain.SourceOrDefault(in.Source)
	if !domain.ValidSource(source) {
		return fmt.Errorf("%w: fonte desconhecida %q", domain.ErrInvalidInput, in.Source)
	}

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

	parser := uc.parsers.For(source)
	acts := parseActs(parser, uc.entities, text)
	number := in.EditionNumber
	if number == "" {
		number = parser.EditionNumber(text)
	}
	g := &domain.Gazette{
		Source:        source,
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
