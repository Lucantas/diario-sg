package http

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterPerClientAndTotal(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	l := newRateLimiter(5, 7, time.Minute, func() time.Time { return now })

	for i := 0; i < 5; i++ {
		if !l.allow("a") {
			t.Fatalf("pedido %d do cliente a deveria passar", i+1)
		}
	}
	if l.allow("a") {
		t.Fatal("sexto pedido do cliente a deveria ser barrado")
	}
	if !l.allow("b") || !l.allow("c") {
		t.Fatal("outros clientes passam enquanto houver folga no total")
	}
	if l.allow("d") {
		t.Fatal("o total da janela é 7")
	}

	now = now.Add(time.Minute)
	if !l.allow("a") {
		t.Fatal("a janela nova libera o cliente a")
	}
}

func TestClientKeyUsesFirstForwardedAddress(t *testing.T) {
	r := httptest.NewRequest("POST", "/v1/reports", nil)
	r.RemoteAddr = "10.0.0.1:5555"
	if got := clientKey(r); got != "10.0.0.1" {
		t.Errorf("sem X-Forwarded-For usa o endereço da conexão, veio %q", got)
	}
	r.Header.Set("X-Forwarded-For", " 200.1.2.3 , 10.0.0.9")
	if got := clientKey(r); got != "200.1.2.3" {
		t.Errorf("veio %q", got)
	}
}
