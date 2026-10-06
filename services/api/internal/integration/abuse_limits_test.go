//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
)

func TestSubscriptionsAreLimitedPerEmailAndPerClient(t *testing.T) {
	srv, _ := newServerFor(t, entityAlertGazette)
	subscribe := func(email, query string) int {
		return postJSON(t, srv.URL+"/v1/subscriptions", fmt.Sprintf(`{"email":%q,"query":%q}`, email, query))
	}

	for i := 0; i < 5; i++ {
		if got := subscribe("alvo@jornal.com", fmt.Sprintf("termo %d", i)); got != http.StatusAccepted {
			t.Fatalf("inscrição %d para o mesmo e-mail: esperava 202, veio %d", i+1, got)
		}
	}
	if got := subscribe("ALVO@jornal.com ", "mais um"); got != http.StatusTooManyRequests {
		t.Fatalf("sexta confirmação para o mesmo e-mail na hora: esperava 429, veio %d", got)
	}
	for i := 0; i < 4; i++ {
		if got := subscribe(fmt.Sprintf("outra%d@jornal.com", i), "merenda"); got != http.StatusAccepted {
			t.Fatalf("inscrição %d de outro e-mail: esperava 202, veio %d", i+1, got)
		}
	}
	if got := subscribe("mais-uma@jornal.com", "merenda"); got != http.StatusTooManyRequests {
		t.Fatalf("décima primeira inscrição do mesmo endereço IP na hora: esperava 429, veio %d", got)
	}
}

func TestReadsAreLimitedPerClient(t *testing.T) {
	srv, _ := newServerFor(t, entityAlertGazette)

	for i := 0; i < 120; i++ {
		r, err := http.Get(srv.URL + "/v1/organs")
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		if r.StatusCode != http.StatusOK {
			t.Fatalf("leitura %d: esperava 200, veio %d", i+1, r.StatusCode)
		}
	}
	r, err := http.Get(srv.URL + "/v1/organs")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusTooManyRequests || r.Header.Get("Retry-After") == "" {
		t.Fatalf("leitura 121 no minuto: esperava 429 com Retry-After, veio %d", r.StatusCode)
	}
	if r, err := http.Get(srv.URL + "/healthz"); err != nil || r.StatusCode != http.StatusOK {
		t.Fatalf("healthz fica fora do limite: %v %v", r, err)
	}
}

func TestExportIsLimitedPerClient(t *testing.T) {
	srv, _ := newServerFor(t, entityAlertGazette)

	for i := 0; i < 10; i++ {
		r, err := http.Get(srv.URL + "/v1/acts/export?format=csv")
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		if r.StatusCode != http.StatusOK {
			t.Fatalf("exportação %d: esperava 200, veio %d", i+1, r.StatusCode)
		}
	}
	r, err := http.Get(srv.URL + "/v1/acts/export?format=csv")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("exportação 11 no minuto: esperava 429, veio %d", r.StatusCode)
	}
}
