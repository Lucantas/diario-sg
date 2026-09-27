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
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/pmsgportal"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/siconfi"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/tce"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const requestTimeout = 10 * time.Minute

func main() {
	log := obs.NewLogger("tce")
	from := flag.Int("from", 0, "primeiro ano (0 = ano anterior)")
	to := flag.Int("to", 0, "último ano (0 = ano corrente)")
	flag.Parse()
	cfg, err := config.Load(config.RolePayments)
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

	src := tce.New(cfg.TCEBaseURL, &http.Client{Timeout: requestTimeout})
	storage := gcp.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcp.TokenSourceFor(cfg.StorageEmulator))
	runs := postgres.NewFetchRunRepo(db)
	failed := false
	start := time.Now()
	run, err := usecase.NewLoadPayments(src, postgres.NewPaymentRepo(db), runs, storage, time.Now).Execute(ctx, *from, *to)
	log.Info("empenhos do TCE-RJ", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga dos empenhos falhou", "error", err)
		failed = true
	}
	start = time.Now()
	run, err = usecase.NewLoadStaff(src, postgres.NewStaffRepo(db), runs, storage, time.Now).Execute(ctx, *from, *to)
	log.Info("pessoal do TCE-RJ", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga do pessoal falhou", "error", err)
		failed = true
	}
	start = time.Now()
	run, err = usecase.NewLoadOversight(src, postgres.NewOversightRepo(db), runs, storage, time.Now).Execute(ctx)
	log.Info("controle do TCE-RJ", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga do controle falhou", "error", err)
		failed = true
	}
	start = time.Now()
	portal := pmsgportal.New(cfg.PMSGPortalURL, &http.Client{Timeout: requestTimeout})
	run, err = usecase.NewLoadMunicipalCommitments(portal, postgres.NewMunicipalCommitmentRepo(db), runs, storage, time.Now).Execute(ctx, *from, *to)
	log.Info("empenhos do portal da Prefeitura", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga dos empenhos do portal falhou", "error", err)
		failed = true
	}
	start = time.Now()
	fiscal := siconfi.New(cfg.SiconfiURL, &http.Client{Timeout: requestTimeout})
	run, err = usecase.NewLoadFiscalTotals(fiscal, postgres.NewFiscalRepo(db), runs, storage, time.Now).Execute(ctx)
	log.Info("totais do SICONFI", "run", run, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("carga dos totais do SICONFI falhou", "error", err)
		failed = true
	}
	if failed {
		os.Exit(1)
	}
}
