package gcp

import "testing"

func TestPublisherImplementsMessagePublisher(t *testing.T) {
	var _ MessagePublisher = NewPublisher("p", "localhost:8085", nil)
}
