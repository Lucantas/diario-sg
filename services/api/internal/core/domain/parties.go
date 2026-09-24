package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Party struct {
	CNPJ       string
	Name       string
	PublicBody string
}

const (
	partyWindow     = 200
	minPartyName    = 3
	maxPartyNameLen = 120
)

var (
	partyCNPJRe  = regexp.MustCompile(`\b\d{2}\.?\d{3}\.?\d{3}/\d{4}-\s?\d{2}\b`)
	cnpjLabelRe  = regexp.MustCompile(`(?i)[\s,;–-]*(?:pessoa\s+jur[íi]dica[^,]*?,\s*)?(?:(?:regularmente\s+)?inscrit[ao]\s+no\s+)?C\.?\s?N\.?\s?P\.?\s?J\.?(?:/MF)?(?:\s*\(MF\))?\.?(?:\s+sob\s+o)?(?:\s+n(?:úmero|[º°.o])*\.?)?\s*[:.]?\s*$`)
	partyStartRe = regexp.MustCompile(`(?i:partes|contratad[ao]|contratante)\s*:\s*|Empresa\s*:\s*|\bempresa\s+|:\s*|;\s*| e (?:a |o )?(?:empresa\s+)?|\d{2}\.?\d{3}\.?\d{3}/\d{4}-\s?\d{2}[,.]?`)
	clauseRe     = regexp.MustCompile(`,\s+\p{Ll}`)
	leadingArtRe = regexp.MustCompile(`^(?:a|o|à)\s+`)
	letterRe     = regexp.MustCompile(`\pL`)
)

func PartiesOf(body string) []Party {
	text := strings.Join(strings.Fields(body), " ")
	var parties []Party
	seen := map[string]bool{}
	for _, loc := range partyCNPJRe.FindAllStringIndex(text, -1) {
		cnpj, ok := NormalizeCNPJ(strings.ReplaceAll(text[loc[0]:loc[1]], " ", ""))
		if !ok || seen[cnpj] {
			continue
		}
		seen[cnpj] = true
		p := Party{CNPJ: cnpj, Name: partyName(windowBefore(text, loc[0]))}
		p.PublicBody, _ = PublicBody(cnpj)
		parties = append(parties, p)
	}
	return parties
}

func windowBefore(text string, end int) string {
	start := max(0, end-partyWindow)
	for start > 0 && start < end && !utf8.RuneStart(text[start]) {
		start++
	}
	w := text[start:end]
	if start > 0 {
		if i := strings.IndexByte(w, ' '); i >= 0 {
			w = w[i+1:]
		}
	}
	return w
}

func partyName(before string) string {
	before = strings.TrimRight(cnpjLabelRe.ReplaceAllString(before, ""), " ,.-–")
	if starts := partyStartRe.FindAllStringIndex(before, -1); len(starts) > 0 {
		before = before[starts[len(starts)-1][1]:]
	}
	if loc := clauseRe.FindStringIndex(before); loc != nil {
		before = before[:loc[0]]
	}
	name := strings.TrimSpace(leadingArtRe.ReplaceAllString(strings.TrimSpace(before), ""))
	if len([]rune(name)) < minPartyName || len([]rune(name)) > maxPartyNameLen || !letterRe.MatchString(name) {
		return ""
	}
	return name
}

type ParliamentaryQuota struct {
	Councillor string
	FullName   string
	Month      string
	ValueCents int64
}

var (
	quotaRe      = regexp.MustCompile(`(?i)apresentada\s+(?:(?:pel[oa](?:\s+a)?|por)\s+)?(?:vereador[a]?\s+)?(.+?)(?:\s*[–-]\s*(?:(?:vereador[a]?|ver\.)\s+)?([^–,]+?))?[\s,–-]*(?:relativ[oa]|referente)\s+ao\s+m[êe]s\s+de\s+(\pL+)\s+de\s+(\d{4}),?\s+no\s+valor\s+de\s+R\$\s?(\d{1,3}(?:\.\d{3})*|\d+),(\d{2})`)
	quotaMarkRe  = regexp.MustCompile(`(?i)CEAPM|atividade\s+parlamentar`)
	monthNumbers = map[string]int{"janeiro": 1, "fevereiro": 2, "março": 3, "marco": 3, "abril": 4, "maio": 5, "junho": 6,
		"julho": 7, "agosto": 8, "setembro": 9, "outubro": 10, "novembro": 11, "dezembro": 12}
)

func ParliamentaryQuotaOf(t ActType, body string) (ParliamentaryQuota, bool) {
	if t != ActPrestacaoContas || !quotaMarkRe.MatchString(body) {
		return ParliamentaryQuota{}, false
	}
	m := quotaRe.FindStringSubmatch(strings.Join(strings.Fields(body), " "))
	if m == nil {
		return ParliamentaryQuota{}, false
	}
	month, ok := monthNumbers[strings.ToLower(m[3])]
	if !ok {
		return ParliamentaryQuota{}, false
	}
	cents, err := strconv.ParseInt(nonDigitRe.ReplaceAllString(m[5], "")+m[6], 10, 64)
	if err != nil {
		return ParliamentaryQuota{}, false
	}
	councillor := strings.TrimSpace(m[2])
	if councillor == "" {
		councillor = m[1]
	}
	return ParliamentaryQuota{Councillor: councillor, FullName: m[1], Month: fmt.Sprintf("%s-%02d", m[4], month), ValueCents: cents}, true
}
