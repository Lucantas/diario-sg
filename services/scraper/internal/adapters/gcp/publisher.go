package gcp

import (
	"context"
	"encoding/json"

	"github.com/seu-usuario/diario-sg/pkg/events"
	gcpclient "github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

type EventPublisher struct {
	client *gcpclient.Publisher
	topic  string
}

func NewEventPublisher(c *gcpclient.Publisher, topic string) *EventPublisher {
	return &EventPublisher{client: c, topic: topic}
}

func (p *EventPublisher) EditionFetched(ctx context.Context, e domain.FetchedEdition) error {
	data, err := json.Marshal(events.GazetteFetched{
		EditionNumber:  e.Number,
		PublishedAt:    e.PublishedAt,
		SourceURL:      e.URL,
		StoragePath:    e.StoragePath,
		ChecksumSHA256: e.ChecksumSHA256,
		FetchedAt:      e.FetchedAt,
	})
	if err != nil {
		return err
	}
	_, err = p.client.Publish(ctx, p.topic, data, map[string]string{events.AttrType: events.TypeGazetteFetched})
	return err
}
