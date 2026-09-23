package parser

import (
	"regexp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

var camara = Regex{
	pageHeader: []*regexp.Regexp{
		regexp.MustCompile(`^PODER LEGISLATIVO$`),
		regexp.MustCompile(`^CÂMARA MUNICIPAL DE SÃO GONÇALO$`),
		regexp.MustCompile(`(?i)^São Gonçalo, \d{1,2}º? de \pL+ de \d{4}\.?$`),
		regexp.MustCompile(`^Ano-\d+ / Edição.*$`),
		regexp.MustCompile(`^DIÁRIO OFICIAL ELETRÔNICO.*D\.O\.E\.?$`),
		regexp.MustCompile(`^LEI MUNICIPAL 855/2018.*$`),
		regexp.MustCompile(`^_{10,}$`),
		regexp.MustCompile(`^Página \d+ de \d+$`),
	},
	withoutOrgans: true,
	lineNoise:     regexp.MustCompile(`^(?:Página \d+ de \d+|Ano-\d+ / Edição.*)$`),
	editionRes:    []*regexp.Regexp{regexp.MustCompile(`Ano-\d+\s*/\s*Edição\s*[–-]?\s*(\d+)`)},
}

func ForSource(source string) Regex {
	if source == domain.SourceDiarioCamara {
		return camara
	}
	return New()
}

type Set struct{}

func (Set) For(source string) ports.ActParser { return ForSource(source) }
