package nostore

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/pkg/gcp"
)

func TestStoreHasNothing(t *testing.T) {
	ctx := context.Background()
	if _, err := (Store{}).Get(ctx, "x"); !errors.Is(err, gcp.ErrObjectNotFound) {
		t.Errorf("Get: %v", err)
	}
	if err := (Store{}).Put(ctx, "x", "text/plain", nil); err != nil {
		t.Errorf("Put: %v", err)
	}
}
