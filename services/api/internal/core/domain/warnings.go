package domain

import (
	"regexp"
	"strings"
)

const (
	WarningNoNumber  = "sem_numero"
	WarningTitleOnly = "so_titulo"
	WarningManyPages = "muitas_paginas"
	WarningManyActs  = "varios_atos_possiveis"

	ManyPagesThreshold = 10
)

var PortariaVerbRe = regexp.MustCompile(`^(?:Nomeia|Nomear|Exonera|Exonerar|Designa|Designar|Torna sem efeito|Tornar sem efeito|Cessar? os efeitos|Declar[ao] vago|Concede|Conceder|Retifica|Retificar|Revoga|Revogar|Dispensa|Dispensar|Prorroga|Prorrogar|Suspende|Suspender|Autoriza|Autorizar|Convoca|Convocar|Cede|Ceder|Averba|Averbar)(?: a pedido| ex officio| de ofício)?:?$`)

const SignaturePattern = `São Gonçalo,\s*\d{1,2}º?\s+de\s+\S+\s+de\s+\d{4}`

var signatureRe = regexp.MustCompile(`(?i)` + SignaturePattern)

type WarningFacts struct {
	Title      string
	TitleOnly  bool
	Signatures int
	PageStart  int
	PageEnd    int
}

func WarningFactsOf(a Act) WarningFacts {
	return WarningFacts{Title: a.Title, TitleOnly: IsTitleOnly(a.Title, a.Body), Signatures: CountSignatures(a.Body),
		PageStart: a.PageStart, PageEnd: a.PageEnd}
}

func (h ActHit) WarningFacts() WarningFacts {
	return WarningFacts{Title: h.Title, TitleOnly: h.TitleOnly, Signatures: h.Signatures, PageStart: h.PageStart, PageEnd: h.PageEnd}
}

func ActWarnings(f WarningFacts) []string {
	warnings := []string{}
	if PortariaVerbRe.MatchString(f.Title) {
		warnings = append(warnings, WarningNoNumber)
	}
	if f.TitleOnly {
		warnings = append(warnings, WarningTitleOnly)
	}
	if f.PageStart > 0 && f.PageEnd-f.PageStart+1 >= ManyPagesThreshold {
		warnings = append(warnings, WarningManyPages)
	}
	if f.Signatures > 1 {
		warnings = append(warnings, WarningManyActs)
	}
	return warnings
}

func CountSignatures(body string) int { return len(signatureRe.FindAllStringIndex(body, -1)) }

func IsTitleOnly(title, body string) bool {
	return strings.TrimSpace(body) == strings.TrimSpace(title)
}
