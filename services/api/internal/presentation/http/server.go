// Package http é a camada de apresentação REST: traduz HTTP <-> casos de uso.
// Não contém regra de negócio.
package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Serve roda o servidor até o contexto ser cancelado e então faz shutdown
// gracioso (o Cloud Run envia SIGTERM e dá 10s).
func Serve(ctx context.Context, addr string, h http.Handler, log *slog.Logger) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Info("servidor ouvindo", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
