package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/pkg/obs"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/sicam"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const (
	requestTimeout = time.Minute
	defaultMax     = 3000
)

func main() {
	log := obs.NewLogger("sicam")
	full := flag.Bool("full", false, "lê todos os processos do sitemap, não só os novos e os abertos")
	max := flag.Int("max", defaultMax, "máximo de páginas nesta rodada (0 = sem limite)")
	flag.Parse()
	cfg, err := config.Load(config.RoleSICAM)
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

	src := sicam.New(cfg.SICAMSiteURL, &http.Client{Timeout: requestTimeout})
	storage := gcp.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))
	uc := usecase.NewLoadBills(src, postgres.NewBillRepo(db), postgres.NewFetchRunRepo(db), storage, time.Now)
	start := time.Now()
	run, err := uc.Execute(ctx, *full, *max)
	log.Info("processos do SICAM", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga dos processos falhou", "error", err)
		os.Exit(1)
	}
}
