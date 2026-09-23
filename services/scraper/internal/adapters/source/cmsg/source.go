package cmsg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

const (
	DefaultURL = "https://www.cmsg.rj.gov.br/diariooficialeletronico/"
	userAgent  = "diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)"
)

type Source struct {
	baseURL *url.URL
	http    *http.Client
	delay   time.Duration
	last    time.Time
}

func New(baseURL string) (*Source, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("SOURCE_URL inválida: %q", baseURL)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 30 * time.Second
	return &Source{baseURL: u, http: &http.Client{Timeout: 120 * time.Second, Transport: transport}, delay: 2 * time.Second}, nil
}

func (s *Source) Name() string { return domain.SourceDiarioCamara }

func (s *Source) editionURL(day time.Time) string {
	return s.baseURL.ResolveReference(&url.URL{Path: "PUBLICACOES/" + day.Format(time.DateOnly) + ".pdf"}).String()
}

func (s *Source) ListEditions(ctx context.Context, from, to time.Time) ([]domain.Edition, error) {
	var out []domain.Edition
	for day := dayStart(from); !day.After(dayStart(to)); day = day.AddDate(0, 0, 1) {
		e := domain.Edition{Source: domain.SourceDiarioCamara, PublishedAt: day, URL: s.editionURL(day)}
		found, err := s.exists(ctx, e.URL)
		if err != nil {
			return nil, err
		}
		if found {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *Source) exists(ctx context.Context, target string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, target, nil)
	if err != nil {
		return false, err
	}
	resp, err := s.send(ctx, req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	}
	return false, fmt.Errorf("HEAD %s: status %d", target, resp.StatusCode)
}

func (s *Source) Download(ctx context.Context, e domain.Edition) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.send(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: status %d", e.URL, resp.StatusCode)
	}
	return resp.Body, nil
}

func (s *Source) send(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := s.throttle(ctx); err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return s.http.Do(req)
}

func (s *Source) throttle(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	wait := s.delay - time.Since(s.last)
	if !s.last.IsZero() && wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.last = time.Now()
	return nil
}

func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
