package parser

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func classify(title, body string) domain.ActType {
	t := strings.ToUpper(title)
	switch {
	case strings.HasPrefix(t, "PORT"), isPortariaVerb(title):
		return classifyPortaria(title, body)
	case strings.HasPrefix(t, "DECRETO"):
		return domain.ActDecreto
	case strings.HasPrefix(t, "LEI"):
		return domain.ActLei
	case hasAnyPrefix(t, "RESOLUÇÃO", "DELIBERAÇÃO", "INSTRUÇÃO NORMATIVA", "ORDEM DE SERVIÇO"):
		return domain.ActResolucao
	case strings.HasPrefix(t, "DESPACHO"):
		return domain.ActDespacho
	case strings.HasPrefix(t, "EDITAL"):
		if containsAny(t, "LICITAÇÃO", "PREGÃO", "CHAMAMENTO", "CONCORRÊNCIA") {
			return domain.ActLicitacao
		}
		return domain.ActEdital
	case strings.HasPrefix(t, "ATA D"):
		return domain.ActAta
	case strings.HasPrefix(t, "EXTRATO"):
		return classifyExtrato(t)
	case strings.HasPrefix(t, "AVISO"):
		if containsAny(t, "DISPENSA", "INEXIGIBILIDADE") {
			return domain.ActDispensa
		}
		return domain.ActLicitacao
	case hasAnyPrefix(t, "CHAMAMENTO", "PREGÃO", "HOMOLOGAÇÃO", "ADJUDICAÇÃO"):
		return domain.ActLicitacao
	case hasAnyPrefix(t, "DISPENSA", "INEXIGIBILIDADE", "RATIFICAÇÃO"):
		return domain.ActDispensa
	case strings.HasPrefix(t, "TERMO ADITIVO"), strings.HasPrefix(t, "APOSTILA"):
		return domain.ActAditivo
	case strings.HasPrefix(t, "CONTRATO"):
		return domain.ActContrato
	case strings.HasPrefix(t, "TERMO DE"):
		if containsAny(t, "PRESTAÇÃO DE CONTAS", "APREENSÃO") {
			return domain.ActOutro
		}
		if containsAny(t, "CONTRATO", "COOPERAÇÃO", "FOMENTO", "COLABORAÇÃO", "CONVÊNIO", "PARCERIA", "ACORDO", "CESSÃO", "COMODATO", "PERMISSÃO") {
			return domain.ActContrato
		}
	}
	return domain.ActOutro
}

func classifyExtrato(t string) domain.ActType {
	switch {
	case containsAny(t, "ADITIVO", "APOSTILA"):
		return domain.ActAditivo
	case containsAny(t, "INEXIGIBILIDADE", "DISPENSA", "RATIFICAÇÃO"):
		return domain.ActDispensa
	case containsAny(t, "REGISTRO DE PREÇOS", "HOMOLOGAÇÃO", "ADJUDICAÇÃO", "PREGÃO", "LICITAÇÃO"):
		return domain.ActLicitacao
	}
	return domain.ActContrato
}

var (
	semEfeitoRe     = regexp.MustCompile(`torna(?:r)? sem efeito`)
	exoneraRe       = regexp.MustCompile(`\bexonera(?:r|ção|d[oa]s?)?\b`)
	nomeiaRe        = regexp.MustCompile(`\bnome(?:ia|ar|ação|ad[oa]s?)\b`)
	verbWindowRunes = 600
)

// classifyPortaria olha o verbo nas primeiras linhas do corpo. "Torna sem
// efeito a nomeação" é portaria, não nomeação, por isso é testado primeiro.
func classifyPortaria(title, body string) domain.ActType {
	rest := strings.TrimPrefix(body, title)
	if utf8.RuneCountInString(rest) > verbWindowRunes {
		rest = string([]rune(rest)[:verbWindowRunes])
	}
	rest = strings.ToLower(rest)
	switch {
	case semEfeitoRe.MatchString(rest):
		return domain.ActPortaria
	case exoneraRe.MatchString(rest):
		return domain.ActExoneracao
	case nomeiaRe.MatchString(rest):
		return domain.ActNomeacao
	}
	return domain.ActPortaria
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
