package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	SourceNorms  = "siapegov_normas"
	normColumns  = 8
	normDayShape = "02/01/2006"
)

type NormKind string

const (
	NormLaw           NormKind = "lei"
	NormComplementary NormKind = "lei_complementar"
	NormOrganic       NormKind = "lei_organica"
	NormDecree        NormKind = "decreto"

	NormResolution        NormKind = "resolucao"
	NormOrganicAmendment  NormKind = "emenda_lei_organica"
	NormLegislativeDecree NormKind = "decreto_legislativo"
)

var NormCategories = map[NormKind]string{NormLaw: "01", NormOrganic: "02", NormComplementary: "03", NormDecree: "05"}

var NormKindsInOrder = []NormKind{NormLaw, NormComplementary, NormOrganic, NormDecree}

var normKindNames = map[NormKind]string{
	NormLaw: "Lei", NormComplementary: "Lei Complementar", NormOrganic: "Lei Orgânica", NormDecree: "Decreto",
	NormResolution: "Resolução", NormOrganicAmendment: "Emenda à Lei Orgânica", NormLegislativeDecree: "Decreto Legislativo",
}

func (k NormKind) Name() string { return normKindNames[k] }

func NormKindFromSICAM(label string) NormKind {
	folded := foldAccents(strings.Join(strings.Fields(label), " "))
	for kind, name := range normKindNames {
		if foldAccents(name) == folded {
			return kind
		}
	}
	return ""
}

type Norm struct {
	Kind          NormKind
	Number        int
	Year          int
	Suffix        string
	Author        string
	Summary       string
	PromulgatedOn *time.Time
	TextURL       string
}

func (n Norm) Label() string {
	return strings.TrimSpace(fmt.Sprintf("%d/%d %s", n.Number, n.Year, n.Suffix))
}

var (
	normNumberRe = regexp.MustCompile(`^\s*0*([\d.]+)\s*/\s*(\d{2}|\d{4})\s*([A-Za-z])?\s*$`)
	controlRe    = regexp.MustCompile(`[\x00-\x1f]`)
	openWindowRe = regexp.MustCompile(`'([^']+)'`)
)

func ParseNormKind(s string) (NormKind, error) {
	k := NormKind(strings.TrimSpace(strings.ToLower(s)))
	if _, ok := NormCategories[k]; !ok {
		return "", fmt.Errorf("%w: tipo de norma %q (use lei, lei_complementar, lei_organica ou decreto)", ErrInvalidInput, s)
	}
	return k, nil
}

func ParseNormNumber(s string) (int, int, string, error) {
	m := normNumberRe.FindStringSubmatch(controlRe.ReplaceAllString(s, ""))
	if m == nil {
		return 0, 0, "", fmt.Errorf("%w: número de norma %q (use número/ano, como 1406/2022)", ErrInvalidInput, s)
	}
	number, err := strconv.Atoi(strings.ReplaceAll(m[1], ".", ""))
	if err != nil || number == 0 {
		return 0, 0, "", fmt.Errorf("%w: número de norma %q", ErrInvalidInput, s)
	}
	year, _ := strconv.Atoi(m[2])
	if len(m[2]) == shortYearDigits {
		year += 2000
		if year > time.Now().Year() {
			year -= 100
		}
	}
	return number, year, strings.ToUpper(m[3]), nil
}

func ParseNorms(kind NormKind, page []byte, base string) ([]Norm, int, error) {
	rows, err := tableRows(page)
	if err != nil {
		return nil, 0, fmt.Errorf("normas %s: %w", kind, err)
	}
	var out []Norm
	invalid := 0
	for _, c := range rows {
		if len(c) != normColumns {
			continue
		}
		number, year, suffix, err := ParseNormNumber(c[0].text)
		if err != nil {
			invalid++
			continue
		}
		n := Norm{Kind: kind, Number: number, Year: year, Suffix: suffix, Author: c[4].text, Summary: c[5].text}
		if t, err := time.Parse(normDayShape, c[6].text); err == nil {
			n.PromulgatedOn = &t
		}
		if m := openWindowRe.FindStringSubmatch(c[7].onclick); m != nil {
			n.TextURL = base + m[1]
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, invalid, fmt.Errorf("normas %s: nenhuma linha na tabela", kind)
	}
	return out, invalid, nil
}

func WithoutRepeatedNorms(norms []Norm) ([]Norm, int) {
	type key struct {
		kind         NormKind
		number, year int
		suffix       string
	}
	seen := make(map[key]bool, len(norms))
	out := make([]Norm, 0, len(norms))
	for _, n := range norms {
		k := key{n.Kind, n.Number, n.Year, n.Suffix}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, n)
	}
	return out, len(norms) - len(out)
}

func NormDiarioSearch(n Norm) string {
	number := strconv.Itoa(n.Number)
	if n.Number >= 1000 {
		number = number[:len(number)-3] + "." + number[len(number)-3:]
	}
	return fmt.Sprintf(`"%s/%d"`, number, n.Year)
}
