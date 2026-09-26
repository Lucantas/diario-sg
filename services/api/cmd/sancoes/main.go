package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/pkg/obs"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/cgu"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const requestTimeout = 5 * time.Minute

func main() {
	log := obs.NewLogger("sancoes")
	cfg, err := config.Load(config.RoleSanctions)
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

	src := cgu.New(cfg.CGUBaseURL, &http.Client{Timeout: requestTimeout})
	storage := gcp.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))
	uc := usecase.NewLoadSanctions(src, postgres.NewSanctionRepo(db), postgres.NewFetchRunRepo(db), storage, time.Now)
	start := time.Now()
	run, err := uc.Execute(ctx)
	log.Info("sanções da CGU", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga falhou", "error", err)
		os.Exit(1)
	}
}
