package tce

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	municipality     = "SAO GONCALO"
	allRows          = "1000000"
	maxResponseBytes = 64 << 20
	attempts         = 3
	defaultRetryWait = 10 * time.Second
)

type Source struct {
	baseURL   string
	client    *http.Client
	retryWait time.Duration
}

func New(baseURL string, client *http.Client) *Source {
	return &Source{baseURL: strings.TrimSuffix(baseURL, "/") + "/", client: client, retryWait: defaultRetryWait}
}

func (s *Source) Commitments(ctx context.Context, year int, each func(header, row []string) error) (string, error) {
	q := url.Values{"ano": {strconv.Itoa(year)}, "municipio": {municipality}, "inicio": {"0"}, "limite": {allRows}, "csv": {"true"}}
	body, err := s.download(ctx, s.baseURL+"empenho_municipio?"+q.Encode())
	if err != nil {
		return "", fmt.Errorf("empenhos de %d: %w", year, err)
	}
	sum := sha256.Sum256(body)
	r := csv.NewReader(bytes.NewReader(body))
	r.Comma, r.LazyQuotes, r.FieldsPerRecord = ';', true, -1
	header, err := r.Read()
	if err != nil {
		return "", fmt.Errorf("cabeçalho dos empenhos de %d: %w", year, err)
	}
	for {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			return hex.EncodeToString(sum[:]), nil
		}
		if err != nil {
			return "", fmt.Errorf("empenhos de %d: %w", year, err)
		}
		if err := each(header, row); err != nil {
			return "", err
		}
	}
}

func (s *Source) Staff(ctx context.Context, year int) ([]byte, error) {
	q := url.Values{"ano": {strconv.Itoa(year)}, "municipio": {municipality}, "inicio": {"0"}, "limite": {allRows}}
	body, err := s.download(ctx, s.baseURL+"situacao_funcional?"+q.Encode())
	if err != nil {
		return nil, fmt.Errorf("situação funcional de %d: %w", year, err)
	}
	return body, nil
}

func (s *Source) Dataset(ctx context.Context, name string) ([]byte, error) {
	body, err := s.download(ctx, s.baseURL+url.PathEscape(name))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return body, nil
}

func (s *Source) download(ctx context.Context, u string) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		body, retry, err := s.get(ctx, u)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(s.retryWait * time.Duration(attempt)):
		}
	}
	return nil, lastErr
}

func (s *Source) get(ctx context.Context, u string) ([]byte, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, false, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode >= http.StatusInternalServerError, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, true, err
	}
	if len(body) > maxResponseBytes {
		return nil, false, fmt.Errorf("resposta maior que %d bytes", maxResponseBytes)
	}
	return body, false, nil
}
