package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Publisher struct {
	baseURL string
	project string
	ts      TokenSource
	http    *http.Client
}

func NewPublisher(project, emulatorHost string, ts TokenSource) *Publisher {
	base := "https://pubsub.googleapis.com"
	if emulatorHost != "" {
		base = "http://" + emulatorHost
	}
	return &Publisher{baseURL: base, project: project, ts: ts, http: &http.Client{Timeout: 15 * time.Second}}
}

type pubsubMessage struct {
	Data       []byte            `json:"data"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

func (p *Publisher) Publish(ctx context.Context, topic string, data []byte, attrs map[string]string) (string, error) {
	payload, err := json.Marshal(map[string]any{"messages": []pubsubMessage{{Data: data, Attributes: attrs}}})
	if err != nil {
		return "", err
	}
	u := fmt.Sprintf("%s/v1/projects/%s/topics/%s:publish", p.baseURL, p.project, topic)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if err := authorize(ctx, p.ts, req); err != nil {
		return "", err
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("pubsub publish %s: status %d: %s", topic, resp.StatusCode, msg)
	}
	var out struct {
		MessageIDs []string `json:"messageIds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || len(out.MessageIDs) == 0 {
		return "", fmt.Errorf("pubsub publish %s: resposta inválida", topic)
	}
	return out.MessageIDs[0], nil
}

type PushEnvelope struct {
	Message struct {
		Data       []byte            `json:"data"`
		Attributes map[string]string `json:"attributes"`
		MessageID  string            `json:"messageId"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}
