package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

var saoPaulo = time.FixedZone("-0300", -3*60*60)

const minuteLayout = "2006-01-02 15:04"

func formatKey(k domain.APIKey) string {
	lastUse := "nunca usada"
	if k.LastUsedAt.Year() > 1970 {
		lastUse = "último uso " + k.LastUsedAt.In(saoPaulo).Format(minuteLayout)
	}
	status := "ativa"
	if k.RevokedAt.Year() > 1970 {
		status = "revogada em " + k.RevokedAt.In(saoPaulo).Format(minuteLayout)
	}
	return strings.Join([]string{
		k.Prefix,
		"criada " + k.CreatedAt.In(saoPaulo).Format(minuteLayout),
		lastUse,
		fmt.Sprintf("%d chamadas em 30 dias", k.RecentCalls),
		status,
	}, "\t")
}
