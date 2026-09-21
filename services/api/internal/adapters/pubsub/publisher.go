// Package pubsub adapta o cliente Pub/Sub à porta EventPublisher do core.
package pubsub

import (
	"context"
	"encoding/json"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/events"
	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

type EventPublisher struct {
	client       *gcp.Publisher
	topicIndexed string
}

func NewEventPublisher(c *gcp.Publisher, topicIndexed string) *EventPublisher {
	return &EventPublisher{client: c, topicIndexed: topicIndexed}
}

func (p *EventPublisher) GazetteIndexed(ctx context.Context, gazetteID string, actsCount int) error {
	data, err := json.Marshal(events.GazetteIndexed{GazetteID: gazetteID, ActsCount: max(actsCount, 0), IndexedAt: time.Now().UTC()})
	if err != nil {
		return err
	}
	_, err = p.client.Publish(ctx, p.topicIndexed, data, map[string]string{events.AttrType: events.TypeGazetteIndexed})
	return err
}
