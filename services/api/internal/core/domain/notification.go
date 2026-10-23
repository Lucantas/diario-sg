package domain

import "errors"

type NotificationClaim int

const (
	ClaimAcquired NotificationClaim = iota
	ClaimAlreadySent
	ClaimInProgress
)

var (
	ErrNotificationInProgress = errors.New("alerta em envio por outra entrega do mesmo evento")
	ErrAlreadyNotified        = errors.New("o provedor já aceitou um e-mail com esta chave de idempotência")
)

func AlertIdempotencyKey(subscriptionID, gazetteID string) string {
	return "alerta/" + subscriptionID + "/" + gazetteID
}
