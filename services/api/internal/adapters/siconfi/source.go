package siconfi

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
	maxResponseBytes = 8 << 20
	defaultPause     = time.Second
	municipalityIBGE = "3304904"
	rreoAnnex01      = "RREO-Anexo 01"
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

func (s *Source) RREOURL(year, period int) string {
	q := url.Values{"an_exercicio": {strconv.Itoa(year)}, "nr_periodo": {strconv.Itoa(period)}, "co_tipo_demonstrativo": {"RREO"},
		"id_ente": {municipalityIBGE}, "no_anexo": {rreoAnnex01}}
	return s.baseURL + "rreo?" + q.Encode()
}

func (s *Source) RREO(ctx context.Context, year, period int) ([]byte, error) {
	if err := s.wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.RREOURL(year, period), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RREO %d/%d: %w", period, year, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RREO %d/%d: status %d", period, year, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("RREO %d/%d: %w", period, year, err)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("RREO %d/%d: resposta maior que %d bytes", period, year, maxResponseBytes)
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
