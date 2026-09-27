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
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/agentes"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/siapegov"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const requestTimeout = 2 * time.Minute

func main() {
	log := obs.NewLogger("agentes")
	from := flag.String("from", "", "primeiro mês, AAAA-MM (vazio = dois meses antes do último)")
	to := flag.String("to", "", "último mês, AAAA-MM (vazio = mês corrente)")
	flag.Parse()
	fromMonth, toMonth, err := months(*from, *to)
	if err != nil {
		log.Error("mês inválido", "error", err)
		os.Exit(2)
	}
	cfg, err := config.Load(config.RoleAgents)
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

	src := agentes.New(cfg.PrefeituraPayURL, cfg.CamaraPayURL, cfg.SICAMURL, &http.Client{Timeout: requestTimeout})
	storage := gcp.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))
	uc := usecase.NewLoadPoliticalAgents(src, postgres.NewPoliticalAgentRepo(db), postgres.NewFetchRunRepo(db), storage, time.Now)
	start := time.Now()
	run, err := uc.Execute(ctx, fromMonth, toMonth)
	log.Info("agentes políticos", "run", run, "duration_ms", time.Since(start).Milliseconds())
	failed := false
	if err != nil {
		log.Error("carga dos agentes políticos falhou", "error", err)
		failed = true
	}
	start = time.Now()
	norms := siapegov.New(cfg.SIAPEGOVURL, &http.Client{Timeout: requestTimeout})
	run, err = usecase.NewLoadNorms(norms, postgres.NewNormRepo(db), postgres.NewFetchRunRepo(db), storage, time.Now).Execute(ctx)
	log.Info("normas do SIAPEGOV", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga das normas falhou", "error", err)
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
		m, err := time.Parse("2006-01", s)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		out[i] = m
	}
	return out[0], out[1], nil
}
