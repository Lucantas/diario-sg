package main

import (
	"context"
	"os"
	"time"

	"github.com/seu-usuario/diario-sg/pkg/obs"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/config"
	"github.com/seu-usuario/diario-sg/services/api/migrations"
)

func main() {
	log := obs.NewLogger("migrate")
	cfg, err := config.Load(config.RoleMigrate)
	if err != nil {
		log.Error("configuração inválida", "error", err)
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("conexão falhou", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	applied, err := postgres.Migrate(ctx, db, migrations.FS)
	if err != nil {
		log.Error("migration falhou", "error", err, "applied", applied)
		os.Exit(1)
	}
	log.Info("migrations aplicadas", "applied", applied)
}
