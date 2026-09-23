package http

import (
	"net/http/httptest"
	"testing"
)

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
