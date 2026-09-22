// Package parser separa o texto de uma edição do Diário Oficial em atos a
// partir dos cabeçalhos usados de fato pela prefeitura (DECRETO Nº, PORTARIA
// - SEI Nº, Port. nº, EXTRATO..., DESPACHO...). É determinístico e auditável;
// os achados que orientam cada regra estão em docs/parser-findings.md.
package parser

import (
	"strings"
	"unicode/utf8"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type Regex struct{}

func New() Regex { return Regex{} }

func (Regex) Parse(text string) []domain.Act {
	lines := joinSplitHeaders(stripPageNoise(splitLines(text)))
	lines = dropPreamble(lines)

	var acts []domain.Act
	var current *segment
	flush := func() {
		if current == nil {
			return
		}
		if act, ok := current.act(len(acts)); ok {
			acts = append(acts, act)
		}
		current = nil
	}

	for i := 0; i < len(lines); i++ {
		l := lines[i]
		switch {
		case isOrganSection(lines, i), isContinuation(l):
			flush()
		case isPortariaTrailer(l):
			if current == nil {
				current = &segment{}
			}
			current.close(l)
			flush()
		case isPortariaVerb(l):
			flush()
			current = &segment{title: l, lines: []string{l}}
		case isHeader(l):
			flush()
			title := []string{l}
			for headerContinues(l) && i+1 < len(lines) && lines[i+1] != "" {
				i++
				l = lines[i]
				title = append(title, l)
			}
			current = &segment{title: strings.Join(title, " "), lines: title}
		default:
			if current == nil {
				if l == "" {
					continue
				}
				current = &segment{title: l, orphan: true}
			}
			current.lines = append(current.lines, l)
		}
	}
	flush()
	return acts
}

type segment struct {
	title  string
	lines  []string
	orphan bool
}

// close dá ao segmento o número de portaria que o encerra.
func (s *segment) close(trailer string) {
	s.title = trailer
	s.orphan = false
	s.lines = append(s.lines, trailer)
}

const (
	maxTitleRunes  = 200
	minOrphanRunes = 40
)

// act monta o ato. Restos sem cabeçalho (carimbos, sobras de tabela antes do
// primeiro ato) só viram ato quando têm conteúdo de verdade.
func (s *segment) act(position int) (domain.Act, bool) {
	body := strings.TrimSpace(strings.Join(s.lines, "\n"))
	if body == "" || (s.orphan && utf8.RuneCountInString(body) < minOrphanRunes) {
		return domain.Act{}, false
	}
	title := strings.Join(strings.Fields(s.title), " ")
	if r := []rune(title); len(r) > maxTitleRunes {
		title = string(r[:maxTitleRunes])
	}
	return domain.Act{Type: classify(title, body), Title: title, Body: body, Position: position}, true
}
