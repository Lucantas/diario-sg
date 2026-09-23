package mcp

import (
	"fmt"
	"strings"
	"time"
)

var abntMonths = [...]string{"jan.", "fev.", "mar.", "abr.", "maio", "jun.", "jul.", "ago.", "set.", "out.", "nov.", "dez."}

var saoPaulo = time.FixedZone("-0300", -3*60*60)

type citable struct {
	GazetteID     string
	Title         string
	EditionNumber string
	PublishedAt   time.Time
	IsExtra       bool
	SourceURL     string
	PageStart     int
	PageEnd       int
	Checksum      string
}

func pageFragment(pageStart int) string {
	if pageStart <= 0 {
		return ""
	}
	return fmt.Sprintf("#page=%d", pageStart)
}

func pageRange(start, end int) string {
	switch {
	case start <= 0:
		return ""
	case end <= start:
		return fmt.Sprint(start)
	default:
		return fmt.Sprintf("%d-%d", start, end)
	}
}

func archivedPDFURL(webURL string, c citable) string {
	return fmt.Sprintf("%s/api/v1/gazettes/%s/pdf%s", strings.TrimRight(webURL, "/"), c.GazetteID, pageFragment(c.PageStart))
}

func abntDate(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), abntMonths[t.Month()-1], t.Year())
}

func sentence(text string) string {
	return strings.TrimRight(text, " \t\n.:;,") + "."
}

func formatCitation(c citable, webURL string, accessed time.Time) string {
	edition := "ed. " + c.EditionNumber
	if c.EditionNumber == "" {
		edition = "ed. s/n"
	}
	if c.IsExtra {
		edition += " (extra)"
	}
	where := []string{edition, abntDate(c.PublishedAt)}
	if pages := pageRange(c.PageStart, c.PageEnd); pages != "" {
		where = append(where, "p. "+pages)
	}
	return strings.Join([]string{
		fmt.Sprintf("SÃO GONÇALO (RJ). Diário Oficial do Município de São Gonçalo, %s.", strings.Join(where, ", ")),
		sentence(c.Title),
		fmt.Sprintf("Disponível em: <%s%s>.", c.SourceURL, pageFragment(c.PageStart)),
		fmt.Sprintf("Cópia arquivada em: <%s>.", archivedPDFURL(webURL, c)),
		fmt.Sprintf("SHA-256 do PDF: %s.", c.Checksum),
		fmt.Sprintf("Acesso em: %s.", abntDate(accessed.In(saoPaulo))),
	}, " ")
}
