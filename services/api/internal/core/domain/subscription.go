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
	Entity           *EntityRef
	Filter           AlertFilter
	Status           SubscriptionStatus
	ConfirmToken     string
	UnsubscribeToken string
	CreatedAt        time.Time
	ConfirmedAt      *time.Time
}

func NewSubscription(email, query string, filter AlertFilter, now time.Time) (Subscription, error) {
	s, err := newPendingSubscription(email, now)
	if err != nil {
		return Subscription{}, err
	}
	s.Query = strings.Join(strings.Fields(query), " ")
	if n := utf8.RuneCountInString(s.Query); (n > 0 || filter.IsZero()) && (n < 3 || n > 200) {
		return Subscription{}, ErrInvalidQuery
	}
	if s.Filter, err = filter.normalized(); err != nil {
		return Subscription{}, err
	}
	return s, nil
}

func NewEntitySubscription(email string, ref EntityRef, now time.Time) (Subscription, error) {
	s, err := newPendingSubscription(email, now)
	if err != nil {
		return Subscription{}, err
	}
	s.Entity = &ref
	return s, nil
}

func (s Subscription) Subject() string {
	if s.Entity != nil {
		return s.Entity.Description()
	}
	var parts []string
	if s.Query != "" {
		parts = append(parts, "“"+s.Query+"”")
	}
	if d := s.Filter.Description(); d != "" {
		parts = append(parts, d)
	}
	return strings.Join(parts, " · ")
}

func newPendingSubscription(email string, now time.Time) (Subscription, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return Subscription{}, ErrInvalidEmail
	}
	confirm, err := newToken()
	if err != nil {
		return Subscription{}, err
	}
	unsub, err := newToken()
	if err != nil {
		return Subscription{}, err
	}
	return Subscription{Email: email, Status: SubscriptionPending, ConfirmToken: confirm,
		UnsubscribeToken: unsub, CreatedAt: now}, nil
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
