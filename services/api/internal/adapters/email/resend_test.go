package email

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func resendAt(t *testing.T, handler http.HandlerFunc) *ResendSender {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	r := NewResendSender("chave", "Diário SG <a@b.c>")
	r.HTTP = &http.Client{Transport: rewriteHost{target: target}}
	return r
}

type rewriteHost struct{ target *url.URL }

func (rw rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme, req.URL.Host = rw.target.Scheme, rw.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func TestResendSendsTheIdempotencyKeyHeader(t *testing.T) {
	var got string
	r := resendAt(t, func(w http.ResponseWriter, req *http.Request) {
		got = req.Header.Get("Idempotency-Key")
		_, _ = io.WriteString(w, `{"id":"1"}`)
	})

	err := r.Send(context.Background(), Message{To: "x@y.z", Subject: "s", HTML: "h", IdempotencyKey: "alerta/s1/g1"})

	if err != nil || got != "alerta/s1/g1" {
		t.Fatalf("header %q, erro %v", got, err)
	}
}

func TestResendOmitsTheHeaderWithoutKey(t *testing.T) {
	present := true
	r := resendAt(t, func(w http.ResponseWriter, req *http.Request) {
		_, present = req.Header["Idempotency-Key"]
		_, _ = io.WriteString(w, `{"id":"1"}`)
	})

	if err := r.Send(context.Background(), Message{To: "x@y.z"}); err != nil || present {
		t.Fatalf("e-mail sem chave não deve mandar o header: presente=%v erro=%v", present, err)
	}
}

func TestResendConflictsOnIdempotencyKey(t *testing.T) {
	cases := map[string]bool{
		"invalid_idempotent_request":     true,
		"concurrent_idempotent_requests": false,
	}
	for name, alreadyNotified := range cases {
		t.Run(name, func(t *testing.T) {
			r := resendAt(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusConflict)
				_, _ = io.WriteString(w, `{"statusCode":409,"name":"`+name+`","message":"x"}`)
			})

			err := r.Send(context.Background(), Message{To: "x@y.z", IdempotencyKey: "alerta/s1/g1"})

			if err == nil || errors.Is(err, domain.ErrAlreadyNotified) != alreadyNotified {
				t.Fatalf("erro inesperado para %s: %v", name, err)
			}
		})
	}
}

func TestSendMatchesForwardsTheIdempotencyKey(t *testing.T) {
	sender := &recSender{}
	n := NewNotifier(sender, "https://site")

	err := n.SendMatches(context.Background(), domain.Subscription{Email: "a@b.c", Query: "x"}, domain.Gazette{}, nil, "alerta/s1/g1")

	if err != nil || sender.sent[0].IdempotencyKey != "alerta/s1/g1" {
		t.Fatalf("chave não chegou ao Sender: %+v %v", sender.sent, err)
	}
}
