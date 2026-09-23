package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterPerClientAndTotal(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	l := New(5, 7, time.Minute, func() time.Time { return now })

	for i := 0; i < 5; i++ {
		if !l.Allow("a") {
			t.Fatalf("pedido %d do cliente a deveria passar", i+1)
		}
	}
	if l.Allow("a") {
		t.Fatal("sexto pedido do cliente a deveria ser barrado")
	}
	if !l.Allow("b") || !l.Allow("c") {
		t.Fatal("outros clientes passam enquanto houver folga no total")
	}
	if l.Allow("d") {
		t.Fatal("o total da janela é 7")
	}

	now = now.Add(time.Minute)
	if !l.Allow("a") {
		t.Fatal("a janela nova libera o cliente a")
	}
}
