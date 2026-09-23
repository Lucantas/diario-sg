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
	lines := dropPreamble(joinSplitHeaders(stripPageNoise(splitLines(text))))

	var acts []domain.Act
	var current *segment
	organ := ""
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
		case isOrganSection(lines, i):
			flush()
			organ = sectionOrgan(l.text, organ)
		case isContinuation(l.text):
			flush()
			organ = ""
		case isPortariaTrailer(l.text):
			if current == nil {
				current = &segment{}
			}
			current.organ = ""
			current.close(l)
			flush()
			organ = ""
		case isPortariaVerb(l.text):
			flush()
			organ = ""
			current = &segment{title: l.text}
			current.add(l)
		case isHeader(l.text):
			flush()
			title := []line{l}
			for headerContinues(l.text) && i+1 < len(lines) && lines[i+1].text != "" {
				i++
				l = lines[i]
				title = append(title, l)
			}
			current = &segment{title: joinTexts(title, " "), organ: organ}
			current.add(title...)
		default:
			if current == nil {
				if l.text == "" {
					continue
				}
				current = &segment{title: l.text, orphan: true, organ: organ}
			}
			current.add(l)
		}
	}
	flush()
	return acts
}

type segment struct {
	title     string
	lines     []string
	orphan    bool
	pageStart int
	pageEnd   int
	organ     string
}

func (s *segment) add(ls ...line) {
	for _, l := range ls {
		s.lines = append(s.lines, l.text)
		if l.text == "" {
			continue
		}
		if s.pageStart == 0 {
			s.pageStart = l.page
		}
		s.pageEnd = l.page
	}
}

// close dá ao segmento o número de portaria que o encerra.
func (s *segment) close(trailer line) {
	s.title = trailer.text
	s.orphan = false
	s.add(trailer)
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
	return domain.Act{Type: classify(title, body), Title: title, Body: body, Position: position,
		PageStart: s.pageStart, PageEnd: s.pageEnd, Organ: s.organ}, true
}

func sectionOrgan(acronym, current string) string {
	switch {
	case nonOrganSections[acronym]:
		return current
	case knownOrgans[acronym]:
		return acronym
	}
	return ""
}
