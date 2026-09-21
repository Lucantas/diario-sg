package events

import (
	"bytes"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInvalidPayloadIsAcked(t *testing.T) {
	h := &PushHandler{Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	body := `{"message":{"data":"` + base64.StdEncoding.EncodeToString([]byte("não é json")) + `","messageId":"1"}}`
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/events/gazette-fetched", bytes.NewBufferString(body)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("payload inválido deve ser confirmado (ack) para não entrar em loop; veio %d", rec.Code)
	}
}
