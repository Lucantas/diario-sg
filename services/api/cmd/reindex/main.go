package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/pkg/obs"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/pdf"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

func main() {
	log := obs.NewLogger("reindex")
	fromFlag := flag.String("from", "", "primeira data (AAAA-MM-DD)")
	toFlag := flag.String("to", "", "última data (AAAA-MM-DD)")
	flag.Parse()
	from, errFrom := time.Parse(time.DateOnly, *fromFlag)
	to, errTo := time.Parse(time.DateOnly, *toFlag)
	if errFrom != nil || errTo != nil {
		log.Error("uso: reindex -from AAAA-MM-DD -to AAAA-MM-DD", "from", *fromFlag, "to", *toFlag)
		os.Exit(2)
	}
	cfg, err := config.Load(config.RoleReindex)
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

	storage := gcp.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))
	uc := usecase.NewReindexGazettes(postgres.NewGazetteRepo(db), storage, pdf.New(), parser.New(), entities.New())
	start := time.Now()
	res, err := uc.Execute(ctx, from, to)
	log.Info("reindexação finalizada", "result", res, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("edições com falha", "error", err)
		os.Exit(1)
	}
}
