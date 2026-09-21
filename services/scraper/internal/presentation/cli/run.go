// Package cli é a camada de apresentação do scraper: ele roda como um
// Cloud Run Job, então a "interface" é a linha de comando e o código de saída.
package cli

import (
	"context"
	"flag"
	"log/slog"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/usecase"
)

// Run interpreta flags, executa o caso de uso e retorna o código de saída.
func Run(ctx context.Context, args []string, uc *usecase.FetchEditions, defaultLookback time.Duration, log *slog.Logger) int {
	fs := flag.NewFlagSet("scraper", flag.ContinueOnError)
	lookback := fs.Duration("lookback", defaultLookback, "janela de busca (ex.: 72h)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	start := time.Now()
	res, err := uc.Execute(ctx, *lookback)
	log.Info("coleta finalizada", "result", res, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("coleta com erros", "error", err)
		return 1 // o Cloud Run Job registra a falha e aplica max_retries
	}
	return 0
}
