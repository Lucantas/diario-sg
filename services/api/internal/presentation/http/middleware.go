package http

import (
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func withMiddleware(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		r.Body = http.MaxBytesReader(rec, r.Body, 1<<20)
		defer func() {
			if p := recover(); p != nil {
				log.Error("panic", "panic", p, "path", r.URL.Path)
				http.Error(rec, `{"error":"erro interno"}`, http.StatusInternalServerError)
			}
			log.Info("request", "method", r.Method, "path", r.URL.Path, "status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"xff", r.Header.Get("X-Forwarded-For"), "true_client_ip", r.Header.Get("True-Client-IP"),
				"cf_connecting_ip", r.Header.Get("CF-Connecting-IP"), "x_real_ip", r.Header.Get("X-Real-IP"),
				"remote_addr", r.RemoteAddr)
		}()
		next.ServeHTTP(rec, r)
	})
}
