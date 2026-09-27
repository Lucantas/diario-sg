package pmsgmural

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	maxResponseBytes = 32 << 20
	defaultPause     = time.Second
	userAgent        = "diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)"
)

//go:embed isrg-root-yr.pem
var isrgRootYR []byte

type Source struct {
	baseURL string
	client  *http.Client
	pause   time.Duration

	mu   sync.Mutex
	last time.Time
}

func New(baseURL string, timeout time.Duration) (*Source, error) {
	roots, err := trustedRoots()
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	return &Source{baseURL: baseURL, client: &http.Client{Timeout: timeout, Transport: transport}, pause: defaultPause}, nil
}

func trustedRoots() (*x509.CertPool, error) {
	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}
	if !roots.AppendCertsFromPEM(isrgRootYR) {
		return nil, errors.New("raiz ISRG Root YR inválida")
	}
	return roots, nil
}

func (s *Source) BaseURL() string { return s.baseURL }

func (s *Source) List(ctx context.Context, name string) ([]byte, error) {
	if err := s.wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+name+".php", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lista %s do mural: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lista %s do mural: status %d", name, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("lista %s do mural: %w", name, err)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("lista %s do mural: resposta maior que %d bytes", name, maxResponseBytes)
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
