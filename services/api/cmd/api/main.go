package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/pkg/obs"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/email"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/pdf"
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

	registry, payments := postgres.NewRegistryRepo(db), postgres.NewPaymentRepo(db)
	sources := usecase.CompanySources{Registry: registry, Sanctions: postgres.NewSanctionRepo(db), Payments: payments, PNCP: postgres.NewPNCPRepo(db), Works: postgres.NewOversightRepo(db), Federal: postgres.NewFederalRepo(db), Municipal: postgres.NewMunicipalCommitmentRepo(db), Mural: postgres.NewProcurementRepo(db)}
	entity := usecase.NewGetEntity(postgres.NewLinkRepo(db), sources)
	search := usecase.NewSearchActs(acts)
	company := usecase.NewGetCompany(acts, sources)
	keys := usecase.NewAPIKeys(postgres.NewAPIKeyRepo(db))
	patterns := usecase.NewListPatterns(postgres.NewPatternRepo(db), postgres.NewSupplierPatternRepo(db))
	webURL := strings.TrimRight(cfg.PublicWebURL, "/")
	mcpHandler := mcpapi.NewHandler(mcpapi.Deps{Search: search, Read: usecase.NewReadAct(gazettes, acts), Entity: entity,
		Group: usecase.NewGroupActs(acts), Page: usecase.NewReadPage(gazettes, storage, pdf.New()),
		Coverage: usecase.NewSourceCoverage(gazettes), Agents: usecase.NewGetPoliticalAgents(postgres.NewPoliticalAgentRepo(db)), Patterns: patterns, Keys: keys, PublicWebURL: cfg.PublicWebURL,
		MetadataURL: webURL + "/.well-known/oauth-protected-resource/api/mcp", Log: l})

	api := &httpapi.API{
		Search:        search,
		Gazette:       usecase.NewGetGazette(gazettes, acts),
		Company:       company,
		Entity:        entity,
		Stats:         usecase.NewActStats(acts),
		Patterns:      patterns,
		Panels:        usecase.NewGetSupplierPanel(postgres.NewPanelRepo(db), registry, payments),
		Staff:         usecase.NewGetStaffPanel(postgres.NewStaffRepo(db), postgres.NewPatternRepo(db)),
		Oversight:     usecase.NewGetOversight(postgres.NewOversightRepo(db), postgres.NewFiscalRepo(db), postgres.NewMunicipalCommitmentRepo(db)),
		Federal:       usecase.NewGetFederal(postgres.NewFederalRepo(db), postgres.NewSpecialTransferRepo(db)),
		Agents:        usecase.NewGetPoliticalAgents(postgres.NewPoliticalAgentRepo(db)),
		Organs:        usecase.NewListOrgans(acts),
		PDF:           usecase.NewGetGazettePDF(gazettes, storage),
		Export:        usecase.NewExportActs(acts),
		Feed:          usecase.NewActFeed(acts),
		Reports:       usecase.NewErrorReports(postgres.NewErrorReportRepo(db)),
		Keys:          keys,
		OAuth:         usecase.NewOAuth(postgres.NewOAuthRepo(db), keys, webURL+"/api/mcp", time.Now),
		MCP:           mcpHandler,
		PublicWebURL:  cfg.PublicWebURL,
		Subscriptions: usecase.NewSubscriptions(postgres.NewSubscriptionRepo(db), notifier),
		Log:           l,
	}
	return httpapi.Serve(ctx, ":"+cfg.Port, api.Routes(), l)
}
