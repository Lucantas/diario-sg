// Ingestão manual de uma edição já baixada. Passa pelo mesmo caso de uso do
// scraper (FetchEditions): grava o PDF no bucket, publica gazette.fetched.v1
// e grava o marcador. Uso: make ingest FILE=edicao.pdf DATE=2026-09-18 [EDITION=1771]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	gcpclient "github.com/seu-usuario/diario-sg/pkg/gcp"
	"github.com/seu-usuario/diario-sg/pkg/obs"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/adapters/gcp"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/adapters/source/localfile"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/adapters/source/pmsg"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/config"
	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/usecase"
)

func main() {
	log := obs.NewLogger("ingest")
	fs := flag.NewFlagSet("ingest", flag.ContinueOnError)
	file := fs.String("file", "", "caminho do PDF da edição")
	date := fs.String("date", "", "data de publicação (AAAA-MM-DD)")
	number := fs.String("edition", "", "número da edição (opcional)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if err := run(*file, *date, *number, log); err != nil {
		log.Error("ingestão falhou", "error", err)
		os.Exit(1)
	}
}

func run(file, date, number string, log *slog.Logger) error {
	if file == "" || date == "" {
		return fmt.Errorf("uso: ingest -file edicao.pdf -date AAAA-MM-DD [-edition N]")
	}
	published, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return fmt.Errorf("data inválida %q: %w", date, err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	base, err := url.Parse(cfg.SourceURL)
	if err != nil {
		return fmt.Errorf("SOURCE_URL inválida: %w", err)
	}
	source, err := localfile.New(file, published, number, pmsg.EditionURL(base, published))
	if err != nil {
		return err
	}
	storage := gcpclient.NewStorage(cfg.Bucket, cfg.StorageEmulator, gcpclient.TokenSourceFor(cfg.StorageEmulator))
	publisher := gcp.NewEventPublisher(
		gcpclient.NewPublisher(cfg.ProjectID, cfg.PubSubEmulatorHost, gcpclient.TokenSourceFor(cfg.PubSubEmulatorHost)),
		cfg.TopicFetched,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	res, err := usecase.NewFetchEditions(source, storage, publisher).Execute(ctx, time.Hour)
	out, _ := json.Marshal(res)
	log.Info("ingestão finalizada", "result", json.RawMessage(out), "file", file, "date", date)
	if err != nil {
		return err
	}
	if res.Skipped == 1 {
		log.Info("edição já ingerida (marcador existe); nada publicado")
	}
	return nil
}
