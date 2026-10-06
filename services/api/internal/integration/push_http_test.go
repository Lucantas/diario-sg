//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	pkgevents "github.com/seu-usuario/diario-sg/pkg/events"
	"github.com/seu-usuario/diario-sg/pkg/pushhttp"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/email"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/pubsub"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
	"github.com/seu-usuario/diario-sg/services/api/internal/presentation/events"
	"github.com/seu-usuario/diario-sg/services/api/migrations"
)

func TestWorkerOverPushHTTPIndexesAndAlertsBeforeReturning(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	ctx := context.Background()
	db := openTestDB(t, ctx, url)
	defer db.Close()
	if _, err := postgres.Migrate(ctx, db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, resetTables); err != nil {
		t.Fatal(err)
	}
	gaz, acts := postgres.NewGazetteRepo(db), postgres.NewActRepo(db)
	subs, nlog := postgres.NewSubscriptionRepo(db), postgres.NewNotificationLog(db)
	box := &inbox{}
	notifier := email.NewNotifier(box, "https://web.exemplo")

	var handler http.Handler
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }))
	defer srv.Close()
	self := pushhttp.New(srv.URL)
	h := &events.PushHandler{
		Index: usecase.NewIndexGazette(gaz, textStore{}, passthroughExtractor{}, parser.Set{}, entities.New(),
			pubsub.NewEventPublisher(self, "gazette-indexed")),
		Match: usecase.NewMatchSubscriptions(gaz, acts, subs, nlog, notifier),
		Runs:  usecase.NewRecordFetchRun(postgres.NewFetchRunRepo(db)),
		Log:   slog.Default(),
	}
	handler = h.Routes()
	confirmSubscription(t, ctx, usecase.NewSubscriptions(subs, notifier), box, "medicamentos")

	data, err := json.Marshal(pkgevents.GazetteFetched{EditionNumber: "1",
		PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC), SourceURL: "https://exemplo/1.pdf",
		StoragePath: "1.pdf", ChecksumSHA256: strings.Repeat("d", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := self.Publish(ctx, "gazette-fetched", data, map[string]string{pkgevents.AttrType: pkgevents.TypeGazetteFetched}); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM acts`).Scan(&count); err != nil || count == 0 {
		t.Fatalf("edição não indexada: %d %v", count, err)
	}
	if len(box.msgs) != 2 {
		t.Fatalf("esperava confirmação + alerta antes do Publish voltar, veio %d e-mails", len(box.msgs))
	}
}

func confirmSubscription(t *testing.T, ctx context.Context, uc *usecase.Subscriptions, box *inbox, query string) {
	t.Helper()
	if _, err := uc.Subscribe(ctx, "rep@jornal.com", query, domain.AlertFilter{}); err != nil {
		t.Fatal(err)
	}
	token := strings.Split(strings.Split(box.msgs[len(box.msgs)-1].HTML, "token=")[1], `"`)[0]
	if _, err := uc.Confirm(ctx, token); err != nil {
		t.Fatal(err)
	}
}
