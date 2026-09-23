package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/pkg/obs"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/email"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/pdf"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/pubsub"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
	"github.com/seu-usuario/diario-sg/services/api/internal/presentation/events"
	httpapi "github.com/seu-usuario/diario-sg/services/api/internal/presentation/http"
)

func main() {
	log := obs.NewLogger("worker")
	if err := run(log); err != nil {
		log.Error("encerrando com erro", "error", err)
		os.Exit(1)
	}
}

func run(l *slog.Logger) error {
	cfg, err := config.Load(config.RoleWorker)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	notifier, err := email.FromConfig(cfg.Notifier, cfg.ResendAPIKey, cfg.EmailFrom, cfg.PublicWebURL, l)
	if err != nil {
		return err
	}
	storage := gcp.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))
	publisher := pubsub.NewEventPublisher(
		gcp.NewPublisher(cfg.ProjectID, cfg.PubSubEmulatorHost, gcp.TokenSourceFor(cfg.PubSubEmulatorHost)),
		cfg.TopicIndexed,
	)
	gazettes := postgres.NewGazetteRepo(db)
	acts := postgres.NewActRepo(db)

	h := &events.PushHandler{
		Index: usecase.NewIndexGazette(gazettes, storage, pdf.New(), parser.New(), entities.New(), publisher),
		Match: usecase.NewMatchSubscriptions(gazettes, acts, postgres.NewSubscriptionRepo(db),
			postgres.NewNotificationLog(db), notifier),
		Log: l,
	}
	return httpapi.Serve(ctx, ":"+cfg.Port, h.Routes(), l)
}
