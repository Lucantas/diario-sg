package usecase

import (
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

func parseActs(p ports.ActParser, x ports.EntityExtractor, text string) []domain.Act {
	acts := p.Parse(text)
	for i := range acts {
		acts[i].Entities = x.Extract(acts[i].Body)
	}
	return acts
}
