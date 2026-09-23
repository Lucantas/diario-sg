package domain

import (
	"crypto/rand"
	"encoding/base64"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

type SubscriptionStatus string

const (
	SubscriptionPending   SubscriptionStatus = "pending"
	SubscriptionActive    SubscriptionStatus = "active"
	SubscriptionCancelled SubscriptionStatus = "cancelled"
)

type Subscription struct {
	ID               string
	Email            string
	Query            string
	Status           SubscriptionStatus
	ConfirmToken     string
	UnsubscribeToken string
	CreatedAt        time.Time
	ConfirmedAt      *time.Time
}

func NewSubscription(email, query string, now time.Time) (Subscription, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return Subscription{}, ErrInvalidEmail
	}
	query = strings.Join(strings.Fields(query), " ")
	if n := utf8.RuneCountInString(query); n < 3 || n > 200 {
		return Subscription{}, ErrInvalidQuery
	}
	confirm, err := newToken()
	if err != nil {
		return Subscription{}, err
	}
	unsub, err := newToken()
	if err != nil {
		return Subscription{}, err
	}
	return Subscription{
		Email:            email,
		Query:            query,
		Status:           SubscriptionPending,
		ConfirmToken:     confirm,
		UnsubscribeToken: unsub,
		CreatedAt:        now,
	}, nil
}

func (s *Subscription) Confirm(now time.Time) error {
	switch s.Status {
	case SubscriptionCancelled:
		return ErrSubscriptionCancelled
	case SubscriptionActive:
		return nil
	}
	s.Status = SubscriptionActive
	s.ConfirmedAt = &now
	return nil
}

func (s *Subscription) Cancel() { s.Status = SubscriptionCancelled }

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
