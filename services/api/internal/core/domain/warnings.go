package domain

import (
	"regexp"
	"strings"
)

const (
	WarningNoNumber  = "sem_numero"
	WarningTitleOnly = "so_titulo"
	WarningManyPages = "muitas_paginas"

	ManyPagesThreshold = 10
)

var PortariaVerbRe = regexp.MustCompile(`^(?:Nomeia|Nomear|Exonera|Exonerar|Designa|Designar|Torna sem efeito|Tornar sem efeito|Cessar? os efeitos|Declar[ao] vago|Concede|Conceder|Retifica|Retificar|Revoga|Revogar|Dispensa|Dispensar|Prorroga|Prorrogar|Suspende|Suspender|Autoriza|Autorizar|Convoca|Convocar|Cede|Ceder|Averba|Averbar)(?: a pedido| ex officio| de ofício)?:?$`)

func ActWarnings(title string, titleOnly bool, pageStart, pageEnd int) []string {
	warnings := []string{}
	if PortariaVerbRe.MatchString(title) {
		warnings = append(warnings, WarningNoNumber)
	}
	if titleOnly {
		warnings = append(warnings, WarningTitleOnly)
	}
	if pageStart > 0 && pageEnd-pageStart+1 >= ManyPagesThreshold {
		warnings = append(warnings, WarningManyPages)
	}
	return warnings
}

func IsTitleOnly(title, body string) bool {
	return strings.TrimSpace(body) == strings.TrimSpace(title)
}
