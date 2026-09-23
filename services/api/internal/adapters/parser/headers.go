package parser

import (
	"regexp"
	"strings"
)

// Cabeçalhos de ato, sempre no início de uma linha em caixa alta. O número
// aparece como Nº, N°, N.º, N. º ou Nº. — o regex num cobre todos.
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

// No anexo de pessoal a portaria abreviada vem FECHANDO o ato: primeiro o
// verbo ("Exonera:", "Nomeia:"), depois o corpo e por último "Port. nº
// 1497/2026". O número, portanto, é o rodapé do segmento corrente.
var portariaTrailerRe = regexp.MustCompile(`^Port\.?\s*n[º°.]?\s*\d+/\d{2,4}`)

// Verbo que abre uma portaria abreviada: a linha inteira é o verbo, com ou
// sem "a pedido" e dois-pontos ("Exonera:", "Nomear:", "Designa",
// "Declaro vago:"). Frases do corpo que começam com o verbo não casam.
var portariaVerbRe = regexp.MustCompile(`^(?:Nomeia|Nomear|Exonera|Exonerar|Designa|Designar|Torna sem efeito|Tornar sem efeito|Cessar? os efeitos|Declar[ao] vago|Concede|Conceder|Retifica|Retificar|Revoga|Revogar|Dispensa|Dispensar|Prorroga|Prorrogar|Suspende|Suspender|Autoriza|Autorizar|Convoca|Convocar|Cede|Ceder|Averba|Averbar)(?: a pedido| ex officio| de ofício)?:?$`)

// "Continuação do D.O.E. em 18/09/2026" abre um novo bloco do anexo de
// pessoal no topo da página; é fronteira de seção, não título de ato.
var continuationRe = regexp.MustCompile(`^Continuação do D\.O\.E\.`)

// isHeader exige caixa alta na linha inteira: palavras de cabeçalho também
// aparecem no meio de frases ("realizará\nDISPENSA DE LICITAÇÃO ... visando a"),
// mas aí a linha traz minúsculas.
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

func isPortariaVerb(line string) bool { return portariaVerbRe.MatchString(line) }

func startsAct(line string) bool {
	return isHeader(line) || isContinuation(line) || isPortariaVerb(line)
}

// Um cabeçalho que termina em preposição/conjunção continua na linha seguinte
// ("EXTRATO DO QUINTO TERMO ADITIVO DE PRORROGAÇÃO AO" + "CONTRATO DE LOCAÇÃO 006/2020.").
var danglingEndRe = regexp.MustCompile(`\b(?:DE|DO|DA|DOS|DAS|AO|À|COM|SEM|E|PARA|NO|NA|NOS|NAS|POR|SOB)$`)

func headerContinues(line string) bool { return danglingEndRe.MatchString(line) }

// Siglas de órgão abrem seção: linha só com letras maiúsculas, seguida de um
// cabeçalho de ato ou do verbo de uma portaria. Não entram em nenhum ato.
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
