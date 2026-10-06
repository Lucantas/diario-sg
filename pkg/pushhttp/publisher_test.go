package pushhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

var _ gcp.MessagePublisher = (*Publisher)(nil)

func TestPublishPostsPushEnvelopeToTopicRoute(t *testing.T) {
	var got gcp.PushEnvelope
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	id, err := New(srv.URL+"/").Publish(context.Background(), "gazette-fetched", []byte(`{"a":1}`), map[string]string{"type": "x"})

	if err != nil {
		t.Fatal(err)
	}
	if path != "/events/gazette-fetched" {
		t.Errorf("rota: %q", path)
	}
	if string(got.Message.Data) != `{"a":1}` || got.Message.Attributes["type"] != "x" {
		t.Errorf("envelope: %+v", got)
	}
	if id == "" || got.Message.MessageID != id {
		t.Errorf("id %q, messageId %q", id, got.Message.MessageID)
	}
	if got.Subscription != "push-http/gazette-fetched" {
		t.Errorf("subscription: %q", got.Subscription)
	}
}

func TestPublishRetriesServerErrorsAndRateLimit(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusInternalServerError)
		case 2:
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	_, err := New(srv.URL).WithWaits(0, 0, 0).Publish(context.Background(), "t", nil, nil)

	if err != nil || calls.Load() != 3 {
		t.Fatalf("err %v, chamadas %d", err, calls.Load())
	}
}

func TestPublishFailsAfterExhaustingRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	_, err := New(srv.URL).WithWaits(0, 0, 0).Publish(context.Background(), "t", nil, nil)

	if err == nil || calls.Load() != 4 {
		t.Fatalf("err %v, chamadas %d", err, calls.Load())
	}
}

func TestPublishDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	_, err := New(srv.URL).WithWaits(0, 0, 0).Publish(context.Background(), "t", nil, nil)

	if err == nil || calls.Load() != 1 {
		t.Fatalf("err %v, chamadas %d", err, calls.Load())
	}
}

func TestPublishRetriesWhenWorkerIsDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	_, err := New(url).WithWaits(0, 0, 0).Publish(context.Background(), "t", nil, nil)

	if err == nil {
		t.Fatal("worker fora do ar deveria virar erro")
	}
}
