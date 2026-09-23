package gcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/events"
	gcpclient "github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

type EventPublisher struct {
	client    *gcpclient.Publisher
	topic     string
	runsTopic string
}

func NewEventPublisher(c *gcpclient.Publisher, topic, runsTopic string) *EventPublisher {
	return &EventPublisher{client: c, topic: topic, runsTopic: runsTopic}
}

func (p *EventPublisher) EditionFetched(ctx context.Context, e domain.FetchedEdition) error {
	data, err := json.Marshal(events.GazetteFetched{
		EditionNumber:  e.Number,
		PublishedAt:    e.PublishedAt,
		SourceURL:      e.URL,
		StoragePath:    e.StoragePath,
		ChecksumSHA256: e.ChecksumSHA256,
		FetchedAt:      e.FetchedAt,
		Source:         e.Source,
	})
	if err != nil {
		return err
	}
	_, err = p.client.Publish(ctx, p.topic, data, map[string]string{events.AttrType: events.TypeGazetteFetched})
	return err
}

func (p *EventPublisher) RunCompleted(ctx context.Context, r domain.FetchRun) error {
	data, err := json.Marshal(events.FetchCompleted{
		RunID:         r.ID,
		Source:        r.Source,
		RequestedFrom: r.RequestedFrom.Format(time.DateOnly),
		RequestedTo:   r.RequestedTo.Format(time.DateOnly),
		Found:         r.Found,
		Stored:        r.Stored,
		Skipped:       r.Skipped,
		Failed:        r.Failed,
		Error:         r.Error,
		StartedAt:     r.StartedAt,
		FinishedAt:    r.FinishedAt,
	})
	if err != nil {
		return err
	}
	_, err = p.client.Publish(ctx, p.runsTopic, data, map[string]string{events.AttrType: events.TypeFetchCompleted})
	return err
}
