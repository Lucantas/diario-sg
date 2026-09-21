// Package parser separa o texto do Diário Oficial em atos usando cabeçalhos
// típicos (DECRETO, PORTARIA, EXTRATO DE CONTRATO...). É uma heurística
// simples e testável; pode ser substituída por ML sem tocar no core.
package parser

import (
	"regexp"
	"strings"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type Regex struct{}

func New() Regex { return Regex{} }

var headerRe = regexp.MustCompile(`(?m)^[ \t]*(DECRETO|PORTARIA|LEI COMPLEMENTAR|LEI|RESOLUÇÃO|EXTRATO D[OE] CONTRATO|EXTRATO D[OE] TERMO ADITIVO|TERMO ADITIVO|AVISO DE LICITAÇÃO|AVISO DE PREGÃO|EDITAL|RATIFICAÇÃO|DISPENSA DE LICITAÇÃO|INEXIGIBILIDADE)\b[^\n]*$`)

func (Regex) Parse(text string) []domain.Act {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	idx := headerRe.FindAllStringIndex(text, -1)
	if len(idx) == 0 {
		if body := strings.TrimSpace(text); body != "" {
			return []domain.Act{{Type: domain.ActOutro, Title: "Edição completa", Body: body}}
		}
		return nil
	}

	var acts []domain.Act
	for i, loc := range idx {
		end := len(text)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		title := strings.Join(strings.Fields(text[loc[0]:loc[1]]), " ")
		body := strings.TrimSpace(text[loc[0]:end])
		acts = append(acts, domain.Act{Type: classify(title, body), Title: title, Body: body, Position: i})
	}
	return acts
}

func classify(title, body string) domain.ActType {
	t := strings.ToUpper(title)
	switch {
	case strings.Contains(t, "ADITIVO"):
		return domain.ActAditivo
	case strings.Contains(t, "CONTRATO"):
		return domain.ActContrato
	case strings.Contains(t, "LICITAÇÃO") && strings.HasPrefix(t, "DISPENSA"),
		strings.HasPrefix(t, "INEXIGIBILIDADE"), strings.HasPrefix(t, "RATIFICAÇÃO"):
		return domain.ActDispensa
	case strings.Contains(t, "LICITAÇÃO"), strings.Contains(t, "PREGÃO"), strings.HasPrefix(t, "EDITAL"):
		return domain.ActLicitacao
	case strings.HasPrefix(t, "LEI"):
		return domain.ActLei
	}

	b := strings.ToUpper(body)
	switch {
	case strings.Contains(b, "EXONERAR"), strings.Contains(b, "EXONERA "):
		return domain.ActExoneracao
	case strings.Contains(b, "NOMEAR"), strings.Contains(b, "NOMEIA "):
		return domain.ActNomeacao
	case strings.HasPrefix(t, "DECRETO"):
		return domain.ActDecreto
	case strings.HasPrefix(t, "PORTARIA"):
		return domain.ActPortaria
	}
	return domain.ActOutro
}
