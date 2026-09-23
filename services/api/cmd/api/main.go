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
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
	httpapi "github.com/seu-usuario/diario-sg/services/api/internal/presentation/http"
	mcpapi "github.com/seu-usuario/diario-sg/services/api/internal/presentation/mcp"
)

func main() {
	log := obs.NewLogger("api")
	if err := run(log); err != nil {
		log.Error("encerrando com erro", "error", err)
		os.Exit(1)
	}
}

func run(l *slog.Logger) error {
	cfg, err := config.Load(config.RoleAPI)
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
	gazettes := postgres.NewGazetteRepo(db)
	acts := postgres.NewActRepo(db)
	storage := gcp.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))

	search := usecase.NewSearchActs(acts)
	company := usecase.NewGetCompany(acts)
	keys := usecase.NewAPIKeys(postgres.NewAPIKeyRepo(db))
	mcpHandler := mcpapi.NewHandler(mcpapi.Deps{Search: search, Read: usecase.NewReadAct(gazettes, acts), Entity: usecase.NewGetEntity(postgres.NewLinkRepo(db)),
		Coverage: usecase.NewSourceCoverage(gazettes), Keys: keys, PublicWebURL: cfg.PublicWebURL, Log: l})

	api := &httpapi.API{
		Search:        search,
		Gazette:       usecase.NewGetGazette(gazettes, acts),
		Company:       company,
		Stats:         usecase.NewActStats(acts),
		Organs:        usecase.NewListOrgans(acts),
		PDF:           usecase.NewGetGazettePDF(gazettes, storage),
		Export:        usecase.NewExportActs(acts),
		Feed:          usecase.NewActFeed(acts),
		Reports:       usecase.NewErrorReports(postgres.NewErrorReportRepo(db)),
		Keys:          keys,
		MCP:           mcpHandler,
		PublicWebURL:  cfg.PublicWebURL,
		Subscriptions: usecase.NewSubscriptions(postgres.NewSubscriptionRepo(db), notifier),
		Log:           l,
	}
	return httpapi.Serve(ctx, ":"+cfg.Port, api.Routes(), l)
}
