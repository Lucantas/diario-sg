package pushhttp

import (
	"context"
	"testing"
)

type fakePubSub struct{}

func (fakePubSub) Publish(context.Context, string, []byte, map[string]string) (string, error) {
	return "ps", nil
}

func TestChooseDefaultsToPubSub(t *testing.T) {
	for _, mode := range []string{"", ModePubSub} {
		got, err := Choose(mode, "", fakePubSub{})
		if _, ok := got.(fakePubSub); err != nil || !ok {
			t.Errorf("%q: %T %v", mode, got, err)
		}
	}
}

func TestChoosePushHTTPNeedsBaseURL(t *testing.T) {
	if _, err := Choose(ModePushHTTP, "", fakePubSub{}); err == nil {
		t.Error("push-http sem PUSH_BASE_URL deveria falhar")
	}
	got, err := Choose(ModePushHTTP, "http://localhost:8081", fakePubSub{})
	if _, ok := got.(*Publisher); err != nil || !ok {
		t.Errorf("%T %v", got, err)
	}
}

func TestChooseRejectsUnknownMode(t *testing.T) {
	if _, err := Choose("pushhttp", "http://x", fakePubSub{}); err == nil {
		t.Error("modo desconhecido deveria falhar")
	}
}
