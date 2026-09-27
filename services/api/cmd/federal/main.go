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
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/cgu"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/transferegov"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const requestTimeout = 10 * time.Minute

func main() {
	log := obs.NewLogger("federal")
	from := flag.String("from", "", "primeiro mês das transferências, AAAAMM (vazio = dois meses antes do último)")
	to := flag.String("to", "", "último mês das transferências, AAAAMM (vazio = último publicado)")
	flag.Parse()
	fromMonth, toMonth, err := months(*from, *to)
	if err != nil {
		log.Error("mês inválido", "error", err)
		os.Exit(2)
	}
	cfg, err := config.Load(config.RoleFederal)
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
	uc := usecase.NewLoadFederal(src, postgres.NewFederalRepo(db), postgres.NewFetchRunRepo(db), storage, time.Now)
	failed := false
	start := time.Now()
	run, err := uc.Amendments(ctx)
	log.Info("emendas parlamentares", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga das emendas falhou", "error", err)
		failed = true
	}
	start = time.Now()
	run, err = uc.Transfers(ctx, fromMonth, toMonth)
	log.Info("transferências federais", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga das transferências falhou", "error", err)
		failed = true
	}
	start = time.Now()
	special := transferegov.New(cfg.TransferegovURL, &http.Client{Timeout: requestTimeout})
	run, err = usecase.NewLoadSpecialTransfers(special, postgres.NewSpecialTransferRepo(db), postgres.NewFetchRunRepo(db), storage, time.Now).Execute(ctx)
	log.Info("transferências especiais", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga das transferências especiais falhou", "error", err)
		failed = true
	}
	if failed {
		os.Exit(1)
	}
}

func months(from, to string) (time.Time, time.Time, error) {
	var out [2]time.Time
	for i, s := range []string{from, to} {
		if s == "" {
			continue
		}
		m, err := time.Parse("200601", s)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		out[i] = m
	}
	return out[0], out[1], nil
}
