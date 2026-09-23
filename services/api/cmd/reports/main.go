package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/seu-usuario/diario-sg/pkg/obs"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

func main() {
	log := obs.NewLogger("reports")
	status := flag.String("status", string(domain.ReportOpen), "aberto, resolvido ou descartado")
	closeID := flag.String("close", "", "id do reporte a fechar")
	closeAs := flag.String("as", "", "resolvido ou descartado (com -close)")
	flag.Parse()

	cfg, err := config.Load(config.RoleReports)
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
	uc := usecase.NewErrorReports(postgres.NewErrorReportRepo(db))

	if *closeID != "" {
		err := uc.Close(ctx, *closeID, domain.ReportStatus(*closeAs))
		switch {
		case errors.Is(err, domain.ErrInvalidReport):
			log.Error("uso: reports -close ID -as resolvido|descartado")
			os.Exit(2)
		case errors.Is(err, domain.ErrNotFound):
			log.Error("reporte não encontrado ou já fechado", "id", *closeID)
			os.Exit(1)
		case err != nil:
			log.Error("fechar reporte falhou", "error", err)
			os.Exit(1)
		}
		log.Info("reporte fechado", "id", *closeID, "status", *closeAs)
		return
	}

	reports, err := uc.List(ctx, domain.ReportStatus(*status))
	if errors.Is(err, domain.ErrInvalidReport) {
		log.Error("status inválido", "status", *status)
		os.Exit(2)
	}
	if err != nil {
		log.Error("listar reportes falhou", "error", err)
		os.Exit(1)
	}
	for _, r := range reports {
		fmt.Println(formatReport(r, cfg.PublicWebURL))
	}
}
