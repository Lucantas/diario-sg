package cli

import (
	"context"
	"flag"
	"log/slog"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/usecase"
)

func Run(ctx context.Context, args []string, uc *usecase.FetchEditions, defaultLookback time.Duration, log *slog.Logger) int {
	fs := flag.NewFlagSet("scraper", flag.ContinueOnError)
	lookback := fs.Duration("lookback", defaultLookback, "janela de busca (ex.: 72h)")
	fromFlag := fs.String("from", "", "backfill: primeira data (AAAA-MM-DD); ignora -lookback")
	toFlag := fs.String("to", "", "backfill: última data (AAAA-MM-DD; padrão hoje)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	start := time.Now()
	var res usecase.FetchResult
	var err error
	if *fromFlag != "" {
		from, to, perr := parsePeriod(*fromFlag, *toFlag)
		if perr != nil {
			log.Error("período inválido", "error", perr)
			return 2
		}
		res, err = uc.ExecuteRange(ctx, from, to)
	} else {
		res, err = uc.Execute(ctx, *lookback)
	}
	log.Info("coleta finalizada", "result", res, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		log.Error("coleta com erros", "error", err)
		return 1
	}
	return 0
}

func parsePeriod(fromS, toS string) (time.Time, time.Time, error) {
	from, err := time.Parse(time.DateOnly, fromS)
	if err != nil {
		return from, from, err
	}
	to := time.Now()
	if toS != "" {
		if to, err = time.Parse(time.DateOnly, toS); err != nil {
			return from, to, err
		}
	}
	return from, to, nil
}
