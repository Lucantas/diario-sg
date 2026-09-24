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

func TestNewEntitySubscription(t *testing.T) {
	ref := EntityRef{Kind: EntityCNPJ, Key: "12345678000190", Label: "12.345.678/0001-90"}

	s, err := NewEntitySubscription(" A@B.com ", ref, time.Now())

	if err != nil || s.Email != "a@b.com" || s.Query != "" || s.Entity == nil || *s.Entity != ref || s.Status != SubscriptionPending {
		t.Fatalf("inscrição inesperada: %+v %v", s, err)
	}
	if s.ConfirmToken == "" || s.ConfirmToken == s.UnsubscribeToken {
		t.Error("tokens devem existir e ser diferentes")
	}
	if _, err := NewEntitySubscription("x", ref, time.Now()); !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("esperava ErrInvalidEmail, veio %v", err)
	}
}

func TestSubscriptionSubject(t *testing.T) {
	termo := Subscription{Query: "merenda escolar"}
	entidade := Subscription{Entity: &EntityRef{Kind: EntityProcesso, Key: "81892025", Label: "8.189/2025"}}

	if got := termo.Subject(); got != "“merenda escolar”" {
		t.Errorf("termo: %q", got)
	}
	if got := entidade.Subject(); got != "processo 8.189/2025" {
		t.Errorf("entidade: %q", got)
	}
}
