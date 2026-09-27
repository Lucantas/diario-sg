package pmsgportal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	maxResponseBytes = 64 << 20
	defaultPause     = time.Second
	userAgent        = "diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)"
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

func (s *Source) Entities(ctx context.Context) ([]byte, error) {
	return s.get(ctx, s.baseURL+"sis_entidade")
}

func (s *Source) Commitments(ctx context.Context, year, entity int) ([]byte, error) {
	q := url.Values{"ano": {strconv.Itoa(year)}, "id_entidade": {strconv.Itoa(entity)}, "dt_inicio_mes": {""}, "dt_final_mes": {""},
		"nome_razao": {""}, "nr_empenho": {""}}
	body, err := s.get(ctx, s.baseURL+"execucao/empenhos/empenhos?"+q.Encode())
	if err != nil {
		return nil, fmt.Errorf("empenhos de %d da entidade %d: %w", year, entity, err)
	}
	return body, nil
}

func (s *Source) get(ctx context.Context, u string) ([]byte, error) {
	if err := s.wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("resposta maior que %d bytes", maxResponseBytes)
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
