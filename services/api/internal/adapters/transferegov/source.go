package transferegov

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	maxResponseBytes = 16 << 20
	defaultPause     = time.Second
	pageLimit        = 1000
)

type Source struct {
	baseURL string
	client  *http.Client
	pause   time.Duration

	mu   sync.Mutex
	last time.Time
}

func New(baseURL string, client *http.Client) *Source {
	return &Source{baseURL: baseURL, client: client, pause: defaultPause}
}

func (s *Source) URL(table string, filter url.Values) string {
	q := url.Values{}
	for k, v := range filter {
		q[k] = v
	}
	q.Set("limit", fmt.Sprint(pageLimit))
	return s.baseURL + table + "?" + q.Encode()
}

func (s *Source) Rows(ctx context.Context, table string, filter url.Values) ([]byte, error) {
	if err := s.wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL(table, filter), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", table, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: status %d", table, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", table, err)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("%s: resposta maior que %d bytes", table, maxResponseBytes)
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%s: %w", table, err)
	}
	if len(rows) >= pageLimit {
		return nil, fmt.Errorf("%s: %d linhas, a resposta pode ter sido cortada", table, len(rows))
	}
	return body, nil
}

func (s *Source) wait(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d := s.pause - time.Since(s.last); d > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
	}
	s.last = time.Now()
	return nil
}
