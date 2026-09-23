package parser

import (
	"regexp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

var camara = Regex{
	pageHeader: regexp.MustCompile(`(?i)^(?:PODER LEGISLATIVO|CÂMARA MUNICIPAL DE SÃO GONÇALO|São Gonçalo, \d{1,2}º? de \pL+ de \d{4}\.?|Ano-\d+ / Edição.*|DIÁRIO OFICIAL ELETRÔNICO.*D\.O\.E\.?|LEI MUNICIPAL 855/2018.*|_{10,}|Página \d+ de \d+)$`),
	lineNoise:  regexp.MustCompile(`^(?:Página \d+ de \d+|Ano-\d+ / Edição.*)$`),
	editionRes: []*regexp.Regexp{regexp.MustCompile(`Ano-\d+\s*/\s*Edição\s*[–-]?\s*(\d+)`)},
}

func ForSource(source string) Regex {
	if source == domain.SourceDiarioCamara {
		return camara
	}
	return New()
}

type Set struct{}

func (Set) For(source string) ports.ActParser { return ForSource(source) }
