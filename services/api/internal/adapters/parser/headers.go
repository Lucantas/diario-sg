package parser

import (
	"regexp"
	"strings"

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
	regexp.MustCompile(`^CONTRATO\s+(?:DE\s+[A-ZÇÃÕÉ]+\s+)?(?:` + num + `\s*)?\d`),
	regexp.MustCompile(`^(?:DISPENSA DE LICITAÇÃO|INEXIGIBILIDADE DE LICITAÇÃO|RATIFICAÇÃO|HOMOLOGAÇÃO|ADJUDICAÇÃO)\b`),
	regexp.MustCompile(`^PREGÃO ELETRÔNICO\b`),
	regexp.MustCompile(`^(?:CONCESSÃO DE LICENÇA|CONVOCAÇÃO|ERRATA|CORRIGENDA|RETIFICAÇÃO|REPUBLICAÇÃO|COMUNICADO|APOSTILA)\b`),
}

var portariaTrailerRe = regexp.MustCompile(`^Port\.?\s*n[º°.]?\s*\d+/\d{2,4}`)

var continuationRe = regexp.MustCompile(`^Continuação do D\.O\.E\.`)

func isHeader(line string) bool {
	line = strings.ReplaceAll(line, "nº", "Nº")
	if strings.ToUpper(line) != line {
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

func startsAct(line string) bool {
	return isHeader(line) || isContinuation(line) || isPortariaVerb(line)
}

var danglingEndRe = regexp.MustCompile(`\b(?:DE|DO|DA|DOS|DAS|AO|À|COM|SEM|E|PARA|NO|NA|NOS|NAS|POR|SOB)$`)

func headerContinues(line string) bool { return danglingEndRe.MatchString(line) }

var organRe = regexp.MustCompile(`^[A-Z]{2,14}$`)

func isOrganSection(lines []line, i int) bool {
	if !organRe.MatchString(lines[i].text) || isHeader(lines[i].text) {
		return false
	}
	for j := i + 1; j < len(lines) && j <= i+2; j++ {
		if lines[j].text == "" {
			continue
		}
		return startsAct(lines[j].text)
	}
	return false
}

var nonOrganSections = map[string]bool{"ANEXO": true}
