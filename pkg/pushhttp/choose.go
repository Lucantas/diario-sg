package pushhttp

import (
	"fmt"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

const (
	ModePubSub   = "pubsub"
	ModePushHTTP = "push-http"
)

func Choose(mode, baseURL string, pubsub gcp.MessagePublisher) (gcp.MessagePublisher, error) {
	switch mode {
	case "", ModePubSub:
		return pubsub, nil
	case ModePushHTTP:
		if baseURL == "" {
			return nil, fmt.Errorf("EVENTS=%s pede PUSH_BASE_URL", ModePushHTTP)
		}
		return New(baseURL), nil
	default:
		return nil, fmt.Errorf("EVENTS desconhecido: %q (use %s ou %s)", mode, ModePubSub, ModePushHTTP)
	}
}
