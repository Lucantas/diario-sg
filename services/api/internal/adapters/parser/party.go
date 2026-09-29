package parser

import (
	"regexp"
	"strings"
)

var partyCNPJRe = regexp.MustCompile(`^CNPJ\b.*\d{2}\.?\d{3}\.?\d{3}/?\d{4}-?\d{2}`)

const partyNamePrefixRunes = 20

func (s *segment) takeParty(nextText string, page int) []line {
	if s == nil {
		return nil
	}
	cnpj := lastTextIndex(s.lines, len(s.lines))
	if cnpj < 0 || !partyCNPJRe.MatchString(s.lines[cnpj]) {
		return nil
	}
	name := lastTextIndex(s.lines, cnpj)
	if name < 0 || !startsWithName(nextText, s.lines[name]) {
		return nil
	}
	party := []line{{text: s.lines[name], page: page}, {text: s.lines[cnpj], page: page}}
	s.lines = s.lines[:name]
	return party
}

func lastTextIndex(lines []string, before int) int {
	for i := before - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return i
		}
	}
	return -1
}

func startsWithName(text, name string) bool {
	name = strings.Join(strings.Fields(name), " ")
	text = strings.Join(strings.Fields(text), " ")
	if r := []rune(name); len(r) > partyNamePrefixRunes {
		name = string(r[:partyNamePrefixRunes])
	}
	return len([]rune(name)) >= 3 && strings.HasPrefix(strings.ToUpper(text), strings.ToUpper(name))
}

func firstTextAfter(lines []line, i int) string {
	for j := i + 1; j < len(lines); j++ {
		if lines[j].text != "" {
			return lines[j].text
		}
	}
	return ""
}
