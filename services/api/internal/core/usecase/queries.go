package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

// SearchActs é a busca pública de atos.
type SearchActs struct{ acts ports.ActRepository }

func NewSearchActs(a ports.ActRepository) *SearchActs { return &SearchActs{acts: a} }

type SearchResult struct {
	Hits   []domain.ActHit
	Total  int
	Limit  int
	Offset int
}

func (uc *SearchActs) Execute(ctx context.Context, f domain.ActFilter) (SearchResult, error) {
	if err := f.Normalize(); err != nil {
		return SearchResult{}, err
	}
	hits, total, err := uc.acts.Search(ctx, f)
	if err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Hits: hits, Total: total, Limit: f.Limit, Offset: f.Offset}, nil
}

// GetGazette retorna uma edição com todos os seus atos.
type GetGazette struct {
	gazettes ports.GazetteRepository
	acts     ports.ActRepository
}

func NewGetGazette(g ports.GazetteRepository, a ports.ActRepository) *GetGazette {
	return &GetGazette{gazettes: g, acts: a}
}

func (uc *GetGazette) Execute(ctx context.Context, id string) (domain.Gazette, []domain.Act, error) {
	g, err := uc.gazettes.FindByID(ctx, id)
	if err != nil {
		return domain.Gazette{}, nil, err
	}
	acts, err := uc.acts.ListByGazette(ctx, id)
	return g, acts, err
}
