package domain

import (
	"regexp"
	"strconv"
	"strings"
)

type PanelValueRole int

const (
	PanelHeadRunes     = 800
	amendmentHeadRunes = 400
)

const (
	PanelValueNone PanelValueRole = iota
	PanelValueContracted
	PanelValueRegistered
	PanelValueAmended
)

var (
	valuelessHeadRe     = regexp.MustCompile(`(?i)homologo\s+o\s+resultado|relat[óo]rio\s+final|apresentados?\s+pelo\s+pregoeiro|^\W*(?:extrato\s+d[aeo]s?\s+)?(?:termo\s+de\s+)?(?:homologa|adjudica|aviso|resultado|multa|san[çc][ãa]o|penalidade|notifica|cancelamento)`)
	priceRegistryHeadRe = regexp.MustCompile(`(?i)^\W*(?:extrato\s+)?(?:d[ae]\s+)?ata\s+(?:[\w./-]+\s+)?de\s+registro|^\W*extrato\s+(?:de\s+publica[çc][ãa]o\s+)?trimestral`)
	contractPartiesRe   = regexp.MustCompile(`(?i)\bpartes\b|part[íi]cipes|\bcontrato\s+n`)
	amendmentHeadRe     = regexp.MustCompile(`(?i)termo\s+aditivo|\baditamento|\bprorroga[çc][ãa]o\s+d[ao]\s+(?:ata|contrato|prazo)|termo\s+de\s+prorroga|objet(?:o|ivo)\b[^.]{0,60}\bprorroga|fica\s+prorrogad`)
	notNewContractingRe = regexp.MustCompile(`(?i)rerratifica|onde\s+se\s+l[êe]|\bleia[- ]se\b|extrato\s+de\s+reajuste|^\W*ata\s+d[aeo]\s+(?:reuni|assembl)|corrigenda|\brevoga[çc][ãa]o|readequa[çc][ãa]o|designa[çc][ãa]o\s+d[aoe]s?\s+(?:servidor|fisca)|para\s+exercer\s+a\s+fiscaliza|termo\s+de\s+permiss[ãa]o|edital\s+de\s+chamamento|presente\s+edital|deste\s+edital|publica[çc][ãa]o\s+do\s+chamamento|pre[âa]mbulo`)
	explicitIncreaseRe  = regexp.MustCompile(`(?i)(?:acr[ée]scimo(?:\s+de\s+valor)?\s+(?:de|no\s+valor\s+de)|valor\s+(?:do\s+)?acr[ée]scimo\s*:?|valor\s+acrescido\s+de)\s*R\$\s?(\d{1,3}(?:\.\d{3})+|\d+),(\d{2})\b`)
	prorrogationRe      = regexp.MustCompile(`(?i)prorroga`)
	withoutCostRe       = regexp.MustCompile(`(?i)sem\s+(?:[ôo]nus|altera[çc][ãa]o\s+(?:d[oe]|n[oa])\s+valor|acr[ée]scimo|reflexo\s+financeiro|custo)`)
	suppressionRe       = regexp.MustCompile(`(?i)supress[ãa]o`)
	directContractingRe = regexp.MustCompile(`(?i)dispensa|inexigibilidade`)
	awardHeadRe         = regexp.MustCompile(`(?i)\bhomologando\b|\bhomologo\b|fica\s+a\s+homologa|determino\s+a\s+homologa`)
)

var valuelessPhases = map[Phase]bool{
	PhaseHomologacao: true, PhaseAditivo: true, PhaseRescisao: true, PhaseAjusteContas: true, PhaseFiscal: true,
}

func PanelValueRoleOf(t ActType, title, head string) PanelValueRole {
	head = strings.TrimSpace(head)
	phase := PhaseOf(t, title)
	opening := firstRunes(head, amendmentHeadRunes)
	if valuelessHeadRe.MatchString(opening) || notNewContractingRe.MatchString(opening) ||
		(!isDirectContracting(t, opening) && awardHeadRe.MatchString(head)) {
		return PanelValueNone
	}
	if t == ActAditivo || phase == PhaseAditivo || amendmentHeadRe.MatchString(opening) {
		return PanelValueAmended
	}
	if valuelessPhases[phase] {
		return PanelValueNone
	}
	if priceRegistryHeadRe.MatchString(head) || (phase == PhaseAtaRegistroPrecos && !contractPartiesRe.MatchString(head)) {
		return PanelValueRegistered
	}
	return PanelValueContracted
}

func isDirectContracting(t ActType, opening string) bool {
	return t == ActDispensa || directContractingRe.MatchString(opening)
}

func AmendmentValueCents(head string, mainValueCents int64, declaredBP int, contractedCents int64) int64 {
	if m := explicitIncreaseRe.FindStringSubmatch(head); m != nil {
		cents, err := strconv.ParseInt(nonDigitRe.ReplaceAllString(m[1], "")+m[2], 10, 64)
		if err == nil {
			return cents
		}
	}
	switch {
	case suppressionRe.MatchString(head):
		return 0
	case prorrogationRe.MatchString(head) && !withoutCostRe.MatchString(head):
		return mainValueCents
	case declaredBP > 0:
		return contractedCents * int64(declaredBP) / (basisPointsPerPoint * 100)
	}
	return 0
}
