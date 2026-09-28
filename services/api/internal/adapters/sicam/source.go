package sicam

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const (
	maxResponseBytes  = 32 << 20
	defaultPause      = time.Second
	userAgent         = "diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)"
	sitemapIndex      = "sitemap.xml"
	processSitemapTag = "sitemap-processos-"
)

type Source struct {
	baseURL string
	client  *http.Client
	pause   time.Duration

	mu   sync.Mutex
	last time.Time
}

func New(baseURL string, client *http.Client) *Source {
	return &Source{baseURL: strings.TrimRight(baseURL, "/") + "/", client: client, pause: defaultPause}
}

func (s *Source) BaseURL() string { return strings.TrimRight(s.baseURL, "/") }

func (s *Source) ProcessKeys(ctx context.Context) ([]domain.BillKey, error) {
	index, err := s.get(ctx, s.baseURL+sitemapIndex)
	if err != nil {
		return nil, err
	}
	sitemaps, err := domain.ParseSitemapIndex(index, processSitemapTag)
	if err != nil {
		return nil, err
	}
	var keys []domain.BillKey
	for _, u := range sitemaps {
		if !s.sameHost(u) {
			continue
		}
		page, err := s.get(ctx, u)
		if err != nil {
			return nil, err
		}
		found, err := domain.ParseProcessSitemap(page)
		if err != nil {
			return nil, err
		}
		keys = append(keys, found...)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("sitemap do SICAM sem processos")
	}
	return keys, nil
}

func (s *Source) ProcessPage(ctx context.Context, key domain.BillKey) ([]byte, error) {
	return s.get(ctx, s.baseURL+"areapublica/processo/"+key.Slug())
}

func (s *Source) sameHost(raw string) bool {
	u, err := url.Parse(raw)
	base, baseErr := url.Parse(s.baseURL)
	return err == nil && baseErr == nil && u.Host == base.Host
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
		return nil, fmt.Errorf("SICAM %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SICAM %s: status %d", u, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("SICAM %s: %w", u, err)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("SICAM %s: resposta maior que %d bytes", u, maxResponseBytes)
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
