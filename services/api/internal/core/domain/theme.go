package domain

import (
	"fmt"
	"regexp"
	"strings"
)

const ThemeEnvironment = "meio_ambiente"

type Theme struct {
	Slug                 string
	Name                 string
	Terms                []string
	Excludes             []string
	Committee            string
	Organs               []string
	PartialOrgan         string
	PartialOrganExcludes []string
	PartialOrganName     string
	Rule                 string

	terms, excludes, partialExcludes, partialName *regexp.Regexp
}

var environmentTheme = newTheme(Theme{
	Slug: ThemeEnvironment,
	Name: "Meio ambiente",
	Terms: []string{
		`meio ambiente`, `\bambienta(l|is)\b`, `arboriz`, `\barvore`, `\bpoda\b`, `supressao (de )?vegeta`, `vegetacao`,
		`florest`, `mata atlantica`, `manguez`, `preservacao permanente`, `protecao permanente`, `unidades? de conservacao`,
		`parque natural`, `\bfauna\b`, `\bflora\b`, `\banimais\b`, `protecao animal`, `\bresiduos?\b`, `\blixo\b`, `reciclag`,
		`coleta seletiva`, `compostag`, `entulho`, `saneamento`, `esgoto`, `drenagem`, `aguas pluviais`, `recursos? hidric`,
		`nascente`, `polui`, `\bruidos?\b`, `licenciamento ambiental`, `mudancas? climatic`, `\bclima\b`, `sustentav`, `agrotox`,
		`queimad`, `desmat`, `energia solar`, `\bplastic`, `\becolog`, `(iptu|telhado|mobilidade urbana|areas?) verde`,
	},
	Excludes:             []string{`taxa de coleta de lixo`, `hepatite`},
	Committee:            "meio ambiente",
	Organs:               []string{"SEMMA", "SEMA", "SEMMADU", "COMMADS", "PROMEA"},
	PartialOrgan:         "SEMMATRAN",
	PartialOrganName:     `meio ambiente e transportes?`,
	PartialOrganExcludes: []string{`corim`, `\btaxi`, `transito`, `transporte`, `estacionamento`, `interdica`, `\bvias?\b`, `onibus`, `mototaxi`, `veiculo`, `\bjari\b`, `\bcadp\b`, `defesa previa`, `\bprevi\b`, `previdencia`},
	Rule: "Meio ambiente: ementa ou título sem acento com um destes termos (" +
		"meio ambiente, ambiental, arborização, árvore, poda, vegetação, floresta, Mata Atlântica, manguezal, preservação permanente, " +
		"unidade de conservação, parque natural, fauna, flora, animais, resíduos, lixo, reciclagem, coleta seletiva, compostagem, " +
		"entulho, saneamento, esgoto, drenagem, águas pluviais, recursos hídricos, nascente, poluição, ruído, licenciamento ambiental, " +
		"clima, sustentável, agrotóxico, queimada, desmatamento, energia solar, plástico, ecologia, IPTU/telhado/área verde), " +
		"exceto taxa de coleta de lixo e hepatite. Proposições também entram quando passaram pela comissão de meio ambiente. " +
		"Atos do Diário também entram pelo órgão: SEMMA, SEMA, SEMMADU, COMMADS e PROMEA; da SEMMATRAN (meio ambiente e transportes), " +
		"só os que, fora o nome da secretaria, não falam de CORIM, táxi, trânsito, transporte, estacionamento, interdição, vias, ônibus, veículos, JARI, CADP, defesa prévia ou previdência.",
})

func newTheme(t Theme) Theme {
	t.terms = regexp.MustCompile(strings.Join(t.Terms, "|"))
	t.excludes = regexp.MustCompile(strings.Join(t.Excludes, "|"))
	t.partialExcludes = regexp.MustCompile(strings.Join(t.PartialOrganExcludes, "|"))
	t.partialName = regexp.MustCompile(t.PartialOrganName)
	return t
}

func ParseTheme(slug string) (Theme, error) {
	switch strings.ToLower(strings.TrimSpace(slug)) {
	case "":
		return Theme{}, nil
	case ThemeEnvironment:
		return environmentTheme, nil
	}
	return Theme{}, fmt.Errorf("%w: tema %q (use %s)", ErrInvalidInput, slug, ThemeEnvironment)
}

func (t Theme) IsZero() bool { return t.Slug == "" }

func (t Theme) MatchesText(s string) bool {
	folded := foldAccents(s)
	return t.terms != nil && t.terms.MatchString(folded) && !t.excludes.MatchString(folded)
}

func (t Theme) MatchesCommittee(name string) bool {
	return t.Committee != "" && strings.Contains(foldAccents(name), t.Committee)
}

func (t Theme) PartialOrganExcluded(text string) bool {
	if t.partialExcludes == nil {
		return false
	}
	return t.partialExcludes.MatchString(t.partialName.ReplaceAllString(foldAccents(text), ""))
}

func (t Theme) TermsSQLRegex() string { return sqlRegex(t.Terms) }

func (t Theme) ExcludeSQLRegex() string { return sqlRegex(t.Excludes) }

func (t Theme) PartialOrganExcludeSQLRegex() string { return sqlRegex(t.PartialOrganExcludes) }

func (t Theme) PartialOrganNameSQLRegex() string { return sqlRegex([]string{t.PartialOrganName}) }

func sqlRegex(parts []string) string {
	return strings.ReplaceAll(strings.Join(parts, "|"), `\b`, `\y`)
}
