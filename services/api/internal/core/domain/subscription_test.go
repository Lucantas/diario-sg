package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewSubscription(t *testing.T) {
	now := time.Now()
	s, err := NewSubscription("  Jornalista@Exemplo.com ", "  secretaria   de saúde ", now)
	if err != nil {
		t.Fatal(err)
	}
	if s.Email != "jornalista@exemplo.com" || s.Query != "secretaria de saúde" || s.Status != SubscriptionPending {
		t.Errorf("normalização inesperada: %+v", s)
	}
	if s.ConfirmToken == "" || s.ConfirmToken == s.UnsubscribeToken {
		t.Error("tokens devem existir e ser diferentes")
	}
	if _, err := NewSubscription("nao-e-email", "abc", now); !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("esperava ErrInvalidEmail, veio %v", err)
	}
	if _, err := NewSubscription("a@b.com", "ab", now); !errors.Is(err, ErrInvalidQuery) {
		t.Errorf("esperava ErrInvalidQuery, veio %v", err)
	}
}

func TestSubscriptionLifecycle(t *testing.T) {
	s, _ := NewSubscription("a@b.com", "licitação", time.Now())
	if err := s.Confirm(time.Now()); err != nil || s.Status != SubscriptionActive {
		t.Fatalf("confirmação falhou: %v %s", err, s.Status)
	}
	s.Cancel()
	if err := s.Confirm(time.Now()); !errors.Is(err, ErrSubscriptionCancelled) {
		t.Errorf("inscrição cancelada não pode ser reconfirmada: %v", err)
	}
}
