package usecase

import (
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

func parseActs(p ports.ActParser, x ports.EntityExtractor, text string) []domain.Act {
	acts := p.Parse(text)
	for i := range acts {
		acts[i].Entities = x.Extract(acts[i].Body)
		acts[i].Modality = domain.ModalityOf(acts[i].Type, acts[i].Title, acts[i].Body)
		acts[i].MainValueCents = domain.MainValueCents(acts[i].Type, acts[i].Body)
		acts[i].NameLines = domain.NameLines(acts[i].Body)
	}
	return acts
}
