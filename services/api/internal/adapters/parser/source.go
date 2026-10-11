package parser

import (
	"regexp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

var camara = Regex{
	pageHeader: []*regexp.Regexp{
		regexp.MustCompile(`^(?:\S{1,12}\s+)?PODER LEGISLATIVO\s*[.º°]?$`),
		regexp.MustCompile(`^C[ÂA]MARA MUNICIPAL DE S[ÃA]O GON[ÇC]ALO$`),
		regexp.MustCompile(`(?i)^São Gonçalo, \d{1,2}º? de \pL+ de \d{4}\.?$`),
		regexp.MustCompile(`^Ano-\d+ / Edição.*$`),
		regexp.MustCompile(`^DI[ÁA]RIO OFICIAL ELETR[ÔO]NICO.*D\.\s*[0O]+\.\s*E\.?$`),
		regexp.MustCompile(`^LEI MUNICIPAL 855/2018.*$`),
		regexp.MustCompile(`^_{10,}$`),
		regexp.MustCompile(`^Página \d+ de \d+$`),
	},
	withoutOrgans:     true,
	joinRepeats:       true,
	splitDecreeTitles: true,
	lineNoise:         regexp.MustCompile(`^(?:Página \d+ de \d+|Ano-\d+ / Edição.*)$`),
	editionRes:        []*regexp.Regexp{regexp.MustCompile(`Ano-\d+\s*/\s*Edição\s*[–-]?\s*(\d+)`)},
}

func ForSource(source string) Regex {
	if source == domain.SourceDiarioCamara {
		return camara
	}
	return New()
}

type Set struct{}

func (Set) For(source string) ports.ActParser { return ForSource(source) }
