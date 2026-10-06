package http

import (
	"net/http/httptest"
	"testing"
)

func TestClientKey(t *testing.T) {
	cases := []struct {
		name, xff string
		hops      int
		want      string
	}{
		{"sem cabeçalho usa a conexão", "", 1, "10.0.0.1"},
		{"hops 0 mantém o primeiro endereço", " 200.1.2.3 , 10.0.0.9", 0, "200.1.2.3"},
		{"hops 1 pega o que o proxy acrescentou", "1.1.1.1, 200.1.2.3", 1, "200.1.2.3"},
		{"hops 2 pula um proxy confiável", "1.1.1.1, 200.1.2.3, 10.0.0.9", 2, "200.1.2.3"},
		{"menos entradas que hops cai na conexão", "200.1.2.3", 3, "10.0.0.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/v1/reports", nil)
			r.RemoteAddr = "10.0.0.1:5555"
			if c.xff != "" {
				r.Header.Set("X-Forwarded-For", c.xff)
			}
			if got := clientKey(r, c.hops); got != c.want {
				t.Errorf("veio %q, queria %q", got, c.want)
			}
		})
	}
}
