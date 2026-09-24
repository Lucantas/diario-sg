package domain

import (
	"strings"
	"unicode/utf8"
)

const (
	mentionSnippetRunes = 280
	mentionLeadRunes    = 120
)

type EntityRef struct {
	Kind  EntityKind
	Key   string
	Label string
}

var entityDescriptionPrefix = map[EntityKind]string{
	EntityCNPJ:     "CNPJ",
	EntityProcesso: "processo",
	EntityContrato: "contrato",
}

func ParseEntityRef(kind EntityKind, value string) (EntityRef, error) {
	if !IsLinkedKind(kind) {
		return EntityRef{}, ErrInvalidInput
	}
	key, err := ParseEntityInput(kind, value)
	if err != nil {
		return EntityRef{}, err
	}
	return EntityRef{Kind: kind, Key: key, Label: entityRefLabel(kind, key, value)}, nil
}

func entityRefLabel(kind EntityKind, key, value string) string {
	switch kind {
	case EntityCNPJ:
		return FormatCNPJ(key)
	case EntityContrato:
		return key
	}
	return EntityLabel(kind, value)
}

func ParseEntityFilter(s string) (*EntityRef, error) {
	if s == "" {
		return nil, nil
	}
	kind, value, ok := strings.Cut(s, ":")
	if !ok {
		return nil, ErrInvalidFilter
	}
	ref, err := ParseEntityRef(EntityKind(kind), value)
	if err != nil {
		return nil, ErrInvalidFilter
	}
	return &ref, nil
}

func (r EntityRef) Description() string {
	return entityDescriptionPrefix[r.Kind] + " " + r.Label
}

func MentionSnippet(body, evidence string) string {
	text := strings.Join(strings.Fields(body), " ")
	needle := strings.Join(strings.Fields(evidence), " ")
	at := strings.Index(text, needle)
	runes := []rune(text)
	if needle == "" || at < 0 {
		return string(runes[:min(len(runes), mentionSnippetRunes)])
	}
	start := utf8.RuneCountInString(text[:at])
	stop := start + utf8.RuneCountInString(needle)
	from := max(0, start-mentionLeadRunes)
	to := max(stop, min(len(runes), from+mentionSnippetRunes))
	return string(runes[from:start]) + "⟦" + string(runes[start:stop]) + "⟧" + string(runes[stop:to])
}
