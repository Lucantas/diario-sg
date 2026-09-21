// Package obs concentra observabilidade compartilhada (logs estruturados).
package obs

import (
	"log/slog"
	"os"
)

// NewLogger cria um logger JSON compatível com o Cloud Logging: os campos
// "severity" e "message" são reconhecidos automaticamente.
func NewLogger(service string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.LevelKey:
				a.Key = "severity"
				if lvl, ok := a.Value.Any().(slog.Level); ok && lvl == slog.LevelWarn {
					a.Value = slog.StringValue("WARNING")
				}
			case slog.MessageKey:
				a.Key = "message"
			}
			return a
		},
	})
	return slog.New(h).With("service", service)
}
