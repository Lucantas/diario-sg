package domain

import (
	"regexp"
	"strings"
)

type PanelValueRole int

const (
	PanelValueNone PanelValueRole = iota
	PanelValueContracted
	PanelValueRegistered
)

var (
	awardTextRe         = regexp.MustCompile(`(?i)homologo\s+o\s+resultado|relat[óo]rio\s+final|apresentados?\s+pelo\s+pregoeiro|^\W*(?:extrato\s+d[aeo]s?\s+)?(?:termo\s+de\s+)?(?:homologa|adjudica|aviso|resultado)`)
	priceRegistryHeadRe = regexp.MustCompile(`(?i)^\W*(?:extrato\s+)?(?:d[ae]\s+)?ata\s+de\s+registro|^\W*extrato\s+trimestral`)
	contractPartiesRe   = regexp.MustCompile(`(?i)\bpartes\b|part[íi]cipes|\bcontrato\s+n`)
)

var valuelessPhases = map[Phase]bool{
	PhaseHomologacao: true, PhaseAditivo: true, PhaseRescisao: true, PhaseAjusteContas: true, PhaseFiscal: true,
}

func PanelValueRoleOf(t ActType, title, head string) PanelValueRole {
	head = strings.TrimSpace(head)
	phase := PhaseOf(t, title)
	if t == ActAditivo || valuelessPhases[phase] || awardTextRe.MatchString(head) {
		return PanelValueNone
	}
	if priceRegistryHeadRe.MatchString(head) || (phase == PhaseAtaRegistroPrecos && !contractPartiesRe.MatchString(head)) {
		return PanelValueRegistered
	}
	return PanelValueContracted
}
