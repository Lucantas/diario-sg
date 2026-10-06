package pushhttp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

type Publisher struct {
	baseURL string
	http    *http.Client
	waits   []time.Duration
}

func New(baseURL string) *Publisher {
	return &Publisher{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Minute},
		waits:   []time.Duration{time.Second, 4 * time.Second, 16 * time.Second},
	}
}

func (p *Publisher) WithWaits(waits ...time.Duration) *Publisher {
	c := *p
	c.waits = waits
	return &c
}

func (p *Publisher) Publish(ctx context.Context, topic string, data []byte, attrs map[string]string) (string, error) {
	id, err := messageID()
	if err != nil {
		return "", err
	}
	var env gcp.PushEnvelope
	env.Message.Data, env.Message.Attributes, env.Message.MessageID = data, attrs, id
	env.Subscription = "push-http/" + topic
	body, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	for attempt := 0; ; attempt++ {
		status, err := p.post(ctx, topic, body)
		if err == nil && status/100 == 2 {
			return id, nil
		}
		if err == nil && status != http.StatusTooManyRequests && status < 500 {
			return "", fmt.Errorf("push %s: status %d", topic, status)
		}
		if attempt >= len(p.waits) {
			return "", fmt.Errorf("push %s: %d tentativas: status %d: %v", topic, attempt+1, status, err)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(p.waits[attempt]):
		}
	}
}

func (p *Publisher) post(ctx context.Context, topic string, body []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/events/"+topic, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.http.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

func messageID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
