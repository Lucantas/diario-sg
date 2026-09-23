package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/pkg/obs"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

func main() {
	log := obs.NewLogger("dump")
	cfg, err := config.Load(config.RoleDump)
	if err != nil {
		log.Error("configuração inválida", "error", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("conexão falhou", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	storage := gcp.NewStorage(cfg.DumpsBucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))
	start := time.Now()
	manifest, err := usecase.NewPublishDump(postgres.NewDumpSource(db), storage).Execute(ctx, start.UTC())
	if err != nil {
		log.Error("dump falhou", "error", err, "duration_ms", time.Since(start).Milliseconds())
		os.Exit(1)
	}
	log.Info("dump publicado", "manifest", manifest, "duration_ms", time.Since(start).Milliseconds())
}
