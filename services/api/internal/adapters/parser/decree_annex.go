package parser

import (
	"regexp"
	"strings"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

var (
	decreeTitleRe  = regexp.MustCompile(`^DECRETO\s+N\s*[º°.]*\s*0*(\d+)\s*/\s*(\d{4})`)
	decreeAnnexRe  = regexp.MustCompile(`^ANEXO\s+(?:AO\s+|DO\s+)?DECRETO\s+N\s*[º°.]*\s*0*(\d+)\s*/\s*(\d{4})`)
	decreeStartsRe = regexp.MustCompile(`^(?:O|A) PREFEIT[OA] MUNICIPAL\b`)
)

type decreeChunk struct {
	owner string
	text  string
}

func decreeKey(m []string) string { return m[1] + "/" + m[2] }

func relocateDecreeAnnexes(acts []domain.Act) []domain.Act {
	index := map[string]int{}
	for i, a := range acts {
		if m := decreeTitleRe.FindStringSubmatch(a.Title); m != nil {
			if _, dup := index[decreeKey(m)]; dup {
				index[decreeKey(m)] = -1
				continue
			}
			index[decreeKey(m)] = i
		}
	}

	bodies := make([]string, len(acts))
	moved := make([][]string, len(acts))
	changed := false
	for i, a := range acts {
		bodies[i] = a.Body
		m := decreeTitleRe.FindStringSubmatch(a.Title)
		if m == nil {
			continue
		}
		own := decreeKey(m)
		var kept []string
		for _, c := range decreeChunks(a.Body, own) {
			if target, ok := index[c.owner]; ok && target >= 0 && c.owner != own {
				moved[target] = append(moved[target], c.text)
				changed = true
				continue
			}
			kept = append(kept, c.text)
		}
		bodies[i] = strings.TrimSpace(strings.Join(kept, "\n"))
	}
	if !changed {
		return acts
	}

	out := make([]domain.Act, len(acts))
	for i, a := range acts {
		a.Body = strings.TrimSpace(strings.Join(append([]string{bodies[i]}, moved[i]...), "\n"))
		a.Type = classify(a.Title, a.Body)
		out[i] = a
	}
	return out
}

func decreeChunks(body, own string) []decreeChunk {
	var chunks []decreeChunk
	var current []string
	kind, owner := "head", own
	preambles := 0
	flush := func() {
		if len(current) > 0 {
			chunks = append(chunks, decreeChunk{owner: owner, text: strings.Join(current, "\n")})
		}
		current = nil
	}
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if m := decreeAnnexRe.FindStringSubmatch(trimmed); m != nil {
			if kind == "preamble" {
				current = append(current, line)
				kind, owner = "annex", decreeKey(m)
				continue
			}
			flush()
			kind, owner = "annex", decreeKey(m)
		} else if decreeStartsRe.MatchString(trimmed) {
			if preambles++; preambles > 1 {
				flush()
				kind, owner = "preamble", own
			}
		}
		current = append(current, line)
	}
	flush()
	return chunks
}
