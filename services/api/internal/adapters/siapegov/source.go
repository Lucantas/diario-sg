package siapegov

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	maxResponseBytes = 64 << 20
	defaultPause     = time.Second
	userAgent        = "diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)"
	resultsPage      = "resulta_leis.php"
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

func (s *Source) BaseURL() string { return s.baseURL }

func (s *Source) Norms(ctx context.Context, category string) ([]byte, error) {
	if err := s.wait(ctx); err != nil {
		return nil, err
	}
	form := url.Values{"Vcategoria": {category}, "Vnumero": {""}, "Vano": {"0"}, "Vementa": {""}, "Vautor": {""}, "paginacao": {"0"},
		"tipo_ordenacao": {"promulgacao"}, "forma_ordenacao": {"desc"}, "cliente": {"pmsaogoncalo"}, "Pesquisar2": {"Pesquisar"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+resultsPage, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("normas da categoria %s: %w", category, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("normas da categoria %s: status %d", category, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("normas da categoria %s: %w", category, err)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("normas da categoria %s: resposta maior que %d bytes", category, maxResponseBytes)
	}
	return latin1ToUTF8(body), nil
}

func latin1ToUTF8(b []byte) []byte {
	out := make([]byte, 0, len(b)+len(b)/8)
	for _, c := range b {
		out = utf8.AppendRune(out, rune(c))
	}
	return out
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
