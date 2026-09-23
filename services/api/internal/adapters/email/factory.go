package email

import (
	"fmt"
	"log/slog"
)

func FromConfig(kind, apiKey, from, webURL string, log *slog.Logger) (*Notifier, error) {
	switch kind {
	case "log", "":
		return NewNotifier(LogSender{Log: log}, webURL), nil
	case "resend":
		return NewNotifier(NewResendSender(apiKey, from), webURL), nil
	default:
		return nil, fmt.Errorf("NOTIFIER desconhecido: %q", kind)
	}
}
