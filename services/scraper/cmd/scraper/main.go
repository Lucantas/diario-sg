package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	gcpclient "github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/pkg/obs"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/adapters/gcp"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/adapters/source/pmsg"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/config"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/usecase"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/presentation/cli"
)

func main() {
	log := obs.NewLogger("scraper")
	cfg, err := config.Load()
	if err != nil {
		log.Error("configuração inválida", "error", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	source, err := pmsg.New(cfg.SourceURL)
	if err != nil {
		log.Error("fonte inválida", "error", err)
		os.Exit(2)
	}
	storage := gcpclient.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcpclient.TokenSourceFor(cfg.StorageEmulator))
	publisher := gcp.NewEventPublisher(
		gcpclient.NewPublisher(cfg.ProjectID, cfg.PubSubEmulatorHost, gcpclient.TokenSourceFor(cfg.PubSubEmulatorHost)),
		cfg.TopicFetched,
		cfg.TopicRuns,
	)

	uc := usecase.NewFetchEditions(source, storage, publisher)
	os.Exit(cli.Run(ctx, os.Args[1:], uc, cfg.Lookback, log))
}
