package gcp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Storage é um cliente mínimo da JSON API do Cloud Storage para um bucket.
type Storage struct {
	baseURL string
	bucket  string
	ts      TokenSource
	http    *http.Client
}

// NewStorage cria o cliente. Se emulatorHost (ex.: "localhost:4443") estiver
// preenchido, fala com o fake-gcs-server via HTTP.
func NewStorage(bucket, emulatorHost string, ts TokenSource) *Storage {
	base := "https://storage.googleapis.com"
	if emulatorHost != "" {
		base = "http://" + emulatorHost
	}
	return &Storage{baseURL: base, bucket: bucket, ts: ts, http: &http.Client{Timeout: 2 * time.Minute}}
}

func (s *Storage) objectURL(name string) string {
	return fmt.Sprintf("%s/storage/v1/b/%s/o/%s", s.baseURL, url.PathEscape(s.bucket), url.PathEscape(name))
}

// Exists retorna true se o objeto existe.
func (s *Storage) Exists(ctx context.Context, name string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.objectURL(name), nil)
	if err != nil {
		return false, err
	}
	if err := authorize(ctx, s.ts, req); err != nil {
		return false, err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("storage exists %q: status %d", name, resp.StatusCode)
	}
}

// Put grava (ou sobrescreve) um objeto.
func (s *Storage) Put(ctx context.Context, name, contentType string, body io.Reader) error {
	u := fmt.Sprintf("%s/upload/storage/v1/b/%s/o?uploadType=media&name=%s",
		s.baseURL, url.PathEscape(s.bucket), url.QueryEscape(name))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	if err := authorize(ctx, s.ts, req); err != nil {
		return err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("storage put %q: status %d: %s", name, resp.StatusCode, msg)
	}
	return nil
}

// Get abre o conteúdo de um objeto. Quem chama deve fechar o reader.
func (s *Storage) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.objectURL(name)+"?alt=media", nil)
	if err != nil {
		return nil, err
	}
	if err := authorize(ctx, s.ts, req); err != nil {
		return nil, err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("storage get %q: status %d", name, resp.StatusCode)
	}
	return resp.Body, nil
}
