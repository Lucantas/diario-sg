package events

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/events"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
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

type recRuns struct{ saved []domain.FetchRun }

func (r *recRuns) Save(_ context.Context, run domain.FetchRun) error {
	r.saved = append(r.saved, run)
	return nil
}

func pushBody(data string) *bytes.Buffer {
	return bytes.NewBufferString(`{"message":{"data":"` + base64.StdEncoding.EncodeToString([]byte(data)) + `","messageId":"7"}}`)
}

func TestFetchCompletedIsRecorded(t *testing.T) {
	repo := &recRuns{}
	h := &PushHandler{Runs: usecase.NewRecordFetchRun(repo), Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	event := `{"run_id":"0b5f3f7e-1c2d-4e5f-8a9b-0c1d2e3f4a5b","source":"diario_prefeitura","requested_from":"2026-09-20",` +
		`"requested_to":"2026-09-23","found":3,"stored":2,"skipped":0,"failed":1,"error":"edição x: timeout",` +
		`"started_at":"2026-09-23T12:00:00Z","finished_at":"2026-09-23T12:01:00Z"}`

	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/events/fetch-completed", pushBody(event)))

	if rec.Code != http.StatusNoContent || len(repo.saved) != 1 {
		t.Fatalf("coleta não gravada: %d %+v", rec.Code, repo.saved)
	}
	got := repo.saved[0]
	if got.Failed != 1 || got.Error != "edição x: timeout" || got.RequestedFrom.Format(time.DateOnly) != "2026-09-20" {
		t.Fatalf("coleta gravada errado: %+v", got)
	}

	for _, bad := range []string{"não é json", `{"run_id":"x","source":"diario_prefeitura","requested_from":"ontem"}`} {
		rec = httptest.NewRecorder()
		h.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/events/fetch-completed", pushBody(bad)))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("payload inválido %q deve ser confirmado, veio %d", bad, rec.Code)
		}
	}
	if len(repo.saved) != 1 {
		t.Fatalf("payload inválido não deveria gravar: %+v", repo.saved)
	}
}

func TestGazetteFetchedCarriesTheSource(t *testing.T) {
	in := indexInputOf(events.GazetteFetched{StoragePath: "raw/diario_camara/2025/11/03/2025-11-03.pdf", Source: "diario_camara"})
	if in.Source != "diario_camara" || in.StoragePath != "raw/diario_camara/2025/11/03/2025-11-03.pdf" {
		t.Errorf("fonte ou caminho perdidos: %+v", in)
	}

	var old events.GazetteFetched
	if err := json.Unmarshal([]byte(`{"published_at":"2026-09-18T00:00:00Z","source_url":"https://do.pmsg.rj.gov.br/x.pdf","storage_path":"gazettes/x.pdf","checksum_sha256":"ab","fetched_at":"2026-09-18T01:00:00Z"}`), &old); err != nil {
		t.Fatal(err)
	}
	if in := indexInputOf(old); in.Source != "" || in.Checksum != "ab" {
		t.Errorf("evento antigo, sem fonte, deveria chegar sem fonte: %+v", in)
	}
}
