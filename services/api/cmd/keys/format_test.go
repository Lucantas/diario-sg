package main

import (
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestFormatKeyShowsUseAndStatus(t *testing.T) {
	k := domain.APIKey{
		Prefix:      "Jtmu7FG4",
		CreatedAt:   time.Date(2026, 9, 23, 17, 43, 0, 0, time.UTC),
		LastUsedAt:  time.Date(2026, 9, 23, 18, 0, 0, 0, time.UTC),
		RecentCalls: 42,
	}

	if got, want := formatKey(k), "Jtmu7FG4\tcriada 2026-09-23 14:43\túltimo uso 2026-09-23 15:00\t42 chamadas em 30 dias\tativa"; got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}

	k.LastUsedAt, k.RecentCalls, k.RevokedAt = time.Time{}, 0, time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	if got, want := formatKey(k), "Jtmu7FG4\tcriada 2026-09-23 14:43\tnunca usada\t0 chamadas em 30 dias\trevogada em 2026-09-24 00:00"; got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}
