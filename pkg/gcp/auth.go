// Package gcp contém clientes REST mínimos para serviços do Google Cloud.
// Usamos a API REST com a biblioteca padrão em vez dos SDKs oficiais para
// manter as imagens pequenas e o código fácil de ler. Todos funcionam também
// contra os emuladores locais (Pub/Sub emulator e fake-gcs-server).
package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// TokenSource fornece tokens OAuth2 de acesso. Uma string vazia significa
// "sem autenticação" (usado com emuladores).
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// NoAuth é usado localmente, contra emuladores.
type NoAuth struct{}

func (NoAuth) Token(context.Context) (string, error) { return "", nil }

// MetadataTokenSource obtém tokens da conta de serviço do Cloud Run pelo
// metadata server, com cache até perto de expirar.
type MetadataTokenSource struct {
	client *http.Client
	mu     sync.Mutex
	token  string
	exp    time.Time
}

func NewMetadataTokenSource() *MetadataTokenSource {
	return &MetadataTokenSource{client: &http.Client{Timeout: 5 * time.Second}}
}

const metadataTokenURL = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token"

func (m *MetadataTokenSource) Token(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.token != "" && time.Until(m.exp) > time.Minute {
		return m.token, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataTokenURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("metadata token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata token: status %d", resp.StatusCode)
	}
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("metadata token: %w", err)
	}
	m.token = body.AccessToken
	m.exp = time.Now().Add(time.Duration(body.ExpiresIn) * time.Second)
	return m.token, nil
}

// TokenSourceFor escolhe NoAuth quando algum emulador está configurado.
func TokenSourceFor(emulatorHost string) TokenSource {
	if emulatorHost != "" {
		return NoAuth{}
	}
	return NewMetadataTokenSource()
}

func authorize(ctx context.Context, ts TokenSource, req *http.Request) error {
	tok, err := ts.Token(ctx)
	if err != nil {
		return err
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	return nil
}
