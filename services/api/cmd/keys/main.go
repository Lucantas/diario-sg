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
	log := obs.NewLogger("keys")
	revoke := flag.String("revoke", "", "prefixo (8 caracteres) da chave a revogar")
	flag.Parse()

	cfg, err := config.Load(config.RoleKeys)
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
	uc := usecase.NewAPIKeys(postgres.NewAPIKeyRepo(db))

	if *revoke != "" {
		err := uc.RevokeByPrefix(ctx, *revoke)
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			log.Error("uso: keys -revoke PREFIXO (os 8 caracteres depois de dsg_)")
			os.Exit(2)
		case errors.Is(err, domain.ErrNotFound):
			log.Error("chave não encontrada ou já revogada", "prefix", *revoke)
			os.Exit(1)
		case err != nil:
			log.Error("revogar chave falhou", "error", err)
			os.Exit(1)
		}
		log.Info("chave revogada", "prefix", *revoke)
		return
	}

	keys, err := uc.List(ctx)
	if err != nil {
		log.Error("listar chaves falhou", "error", err)
		os.Exit(1)
	}
	for _, k := range keys {
		fmt.Println(formatKey(k))
	}
}
