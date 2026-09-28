package parser

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const num = `N\s*[º°.]`

var headerRes = []*regexp.Regexp{
	regexp.MustCompile(`^(?:DECRETO|PORTARIA|LEI COMPLEMENTAR|LEI|RESOLUÇÃO|DELIBERAÇÃO|INSTRUÇÃO NORMATIVA|ORDEM DE SERVIÇO)(?:\s+[A-Z][A-Z/]{1,12}|\s+“[A-Z]”)?\s*(?:[-–]\s*)?(?:SEI\s*)?` + num),
	regexp.MustCompile(`^EXTRATO\b`),
	regexp.MustCompile(`^AVISO\b`),
	regexp.MustCompile(`^EDITAL\b`),
	regexp.MustCompile(`^DESPACHO\b`),
	regexp.MustCompile(`^TERMO (?:DE|ADITIVO)\b`),
	regexp.MustCompile(`^ATA D[AEO]\b`),
	regexp.MustCompile(`^CHAMAMENTO PÚBLICO\b`),
	regexp.MustCompile(`^NOTIFICAÇÃO\b`),
	regexp.MustCompile(`^AUTO DE INFRAÇÃO\b`),
	regexp.MustCompile(`^DESIGNAÇÃO DE FISCA(?:L|IS)\b`),
	regexp.MustCompile(`^AUTORIZAÇÃO D[AE] DESPESA\b`),
	regexp.MustCompile(`^CONTRATO\s+(?:DE\s+[A-ZÇÃÕÉ]+\s+)?(?:` + num + `\s*)?\d`),
	regexp.MustCompile(`^(?:DISPENSA DE LICITAÇÃO|INEXIGIBILIDADE DE LICITAÇÃO|RATIFICAÇÃO|HOMOLOGAÇÃO|ADJUDICAÇÃO)\b`),
	regexp.MustCompile(`^PREGÃO ELETRÔNICO\b`),
	regexp.MustCompile(`^(?:CONCESSÃO DE LICENÇA|CONVOCAÇÃO|ERRATA|CORRIGENDA|RETIFICAÇÃO|REPUBLICAÇÃO|COMUNICADO|APOSTILA)\b`),
}

var notAnActExtractRe = regexp.MustCompile(`^EXTRATO\s+(?:DE\s+)?(?:TOMATE|MALTE|ALOE|BANCÁRIO)\b`)

var portariaTrailerRe = regexp.MustCompile(`^Port\.?\s*n[º°.]?\s*\d+/\d{2,4}`)

var continuationRe = regexp.MustCompile(`^Continuação do D\.O\.E\.`)

var civilDefenseNoticeRe = regexp.MustCompile(`^A COORDENADORIA MUNICIPAL DE DEFESA CIVIL\b`)

const (
	civilDefenseNoticeTitle = "NOTIFICAÇÃO DA DEFESA CIVIL"
	civilDefenseOrgan       = "COMDEC"
)

func withoutAccentTypos(s string) string { return strings.ReplaceAll(s, "ÇÂO", "ÇÃO") }

func isHeader(line string) bool {
	line = withoutAccentTypos(strings.ReplaceAll(line, "nº", "Nº"))
	if strings.ToUpper(line) != line || notAnActExtractRe.MatchString(line) {
		return false
	}
	for _, re := range headerRes {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

func isContinuation(line string) bool { return continuationRe.MatchString(line) }

func isPortariaTrailer(line string) bool { return portariaTrailerRe.MatchString(line) }

func isPortariaVerb(line string) bool { return domain.PortariaVerbRe.MatchString(line) }

func isCivilDefenseNotice(line string) bool { return civilDefenseNoticeRe.MatchString(line) }

func startsAct(line string) bool {
	return isHeader(line) || isContinuation(line) || isPortariaVerb(line)
}

var danglingEndRe = regexp.MustCompile(`\b(?:DE|DO|DA|DOS|DAS|AO|À|COM|SEM|E|PARA|NO|NA|NOS|NAS|POR|SOB)$`)

func headerContinues(line string) bool { return danglingEndRe.MatchString(line) }

var organRe = regexp.MustCompile(`^[A-Z]{2,14}(?:-[A-Z]{2,10})?$`)

func isOrganSection(lines []line, i int) bool {
	if !organRe.MatchString(lines[i].text) || isHeader(lines[i].text) {
		return false
	}
	for j := i + 1; j < len(lines); j++ {
		if lines[j].text != "" {
			return startsAct(lines[j].text) || (domain.IsKnownOrgan(lines[i].text) && isSectionTitle(lines[j].text) && proseFollows(lines, j))
		}
	}
	return false
}

const (
	proseLookahead = 3
	proseMinRunes  = 40
)

func proseFollows(lines []line, title int) bool {
	seen := 0
	for _, l := range lines[title+1:] {
		if l.text == "" {
			continue
		}
		if strings.ToUpper(l.text) != l.text || utf8.RuneCountInString(l.text) >= proseMinRunes {
			return true
		}
		if seen++; seen == proseLookahead {
			return false
		}
	}
	return false
}

const minSectionTitleWords = 2

var sectionTitleWordRe = regexp.MustCompile(`\p{Lu}{3,}`)

func isSectionTitle(line string) bool {
	if strings.ToUpper(line) != line || !unicode.IsLetter([]rune(line)[0]) {
		return false
	}
	words := 0
	for _, w := range sectionTitleWordRe.FindAllString(line, -1) {
		if !domain.IsKnownOrgan(w) {
			words++
		}
	}
	return words >= minSectionTitleWords
}

var nonOrganSections = map[string]bool{"ANEXO": true}
