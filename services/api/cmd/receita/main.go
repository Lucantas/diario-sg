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
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/receita"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const requestTimeout = 10 * time.Minute

func main() {
	log := obs.NewLogger("receita")
	month := flag.String("month", "", "mês publicado pela Receita (AAAA-MM); vazio = o mais recente")
	flag.Parse()
	cfg, err := config.Load(config.RoleReceita)
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

	src := receita.New(cfg.ReceitaBaseURL, cfg.ReceitaShareToken, &http.Client{Timeout: requestTimeout})
	storage := gcp.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))
	uc := usecase.NewLoadRegistry(src, postgres.NewRegistryRepo(db), postgres.NewFetchRunRepo(db), storage, time.Now)
	start := time.Now()
	run, err := uc.Execute(ctx, *month)
	log.Info("cadastro da Receita", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga falhou", "error", err)
		os.Exit(1)
	}
	if err := postgres.RefreshPartnerAppointments(ctx, db); err != nil {
		log.Error("atualização dos nomes de sócios em nomeações falhou", "error", err)
		os.Exit(1)
	}
}
