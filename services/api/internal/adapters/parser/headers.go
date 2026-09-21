package parser

import (
	"regexp"
	"strings"
)

// Cabeçalhos de ato, sempre no início de uma linha em caixa alta. O número
// aparece como Nº, N°, N.º, N. º ou Nº. — o regex num cobre todos.
const num = `N\s*[º°.]`

var headerRes = []*regexp.Regexp{
	regexp.MustCompile(`^(?:DECRETO|PORTARIA|LEI COMPLEMENTAR|LEI|RESOLUÇÃO|DELIBERAÇÃO|INSTRUÇÃO NORMATIVA|ORDEM DE SERVIÇO)(?:\s+[A-Z][A-Z/]{1,12})?\s*(?:[-–]\s*)?(?:SEI\s*)?` + num),
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

// Portaria abreviada do anexo de pessoal: "Port. nº 1497/2026".
var shortPortariaRe = regexp.MustCompile(`^Port\.?\s*n[º°.]?\s*\d+/\d{2,4}`)

var continuationRe = regexp.MustCompile(`^Continuação do D\.O\.E\.`)

// isHeader exige caixa alta na linha inteira: palavras de cabeçalho também
// aparecem no meio de frases ("realizará\nDISPENSA DE LICITAÇÃO ... visando a"),
// mas aí a linha traz minúsculas.
func isHeader(line string) bool {
	if shortPortariaRe.MatchString(line) {
		return true
	}
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

// Um cabeçalho que termina em preposição/conjunção continua na linha seguinte
// ("EXTRATO DO QUINTO TERMO ADITIVO DE PRORROGAÇÃO AO" + "CONTRATO DE LOCAÇÃO 006/2020.").
var danglingEndRe = regexp.MustCompile(`\b(?:DE|DO|DA|DOS|DAS|AO|À|COM|SEM|E|PARA|NO|NA|NOS|NAS|POR|SOB)$`)

func headerContinues(line string) bool { return danglingEndRe.MatchString(line) }

// Siglas de órgão abrem seção: linha só com letras maiúsculas, seguida de um
// cabeçalho de ato. Não entram em nenhum ato.
var organRe = regexp.MustCompile(`^[A-Z]{2,14}$`)

func isOrganSection(lines []string, i int) bool {
	if !organRe.MatchString(lines[i]) || isHeader(lines[i]) {
		return false
	}
	for j := i + 1; j < len(lines) && j <= i+2; j++ {
		if lines[j] == "" {
			continue
		}
		return isHeader(lines[j])
	}
	return false
}
