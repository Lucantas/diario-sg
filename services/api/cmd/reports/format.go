package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func formatReport(r domain.ErrorReport, webURL string) string {
	link := fmt.Sprintf("%s/api/v1/gazettes/%s/pdf", strings.TrimRight(webURL, "/"), r.GazetteID)
	if r.PageStart > 0 {
		link += fmt.Sprintf("#page=%d", r.PageStart)
	}
	message := strings.Join(strings.Fields(r.Message), " ")
	if message == "" {
		message = "(sem descrição)"
	}
	return strings.Join([]string{
		r.ID,
		r.CreatedAt.In(saoPaulo).Format("2006-01-02 15:04"),
		fmt.Sprintf("edição %s de %s, ato %d", r.EditionNumber, r.PublishedAt.Format("02/01/2006"), r.Position),
		string(r.Kind),
		r.ActTitle,
		message,
		link,
	}, "\t")
}

var saoPaulo = time.FixedZone("-0300", -3*60*60)
