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

// GetCompany monta a linha do tempo de um CNPJ.
type GetCompany struct{ acts ports.ActRepository }

func NewGetCompany(a ports.ActRepository) *GetCompany { return &GetCompany{acts: a} }

func (uc *GetCompany) Execute(ctx context.Context, cnpj string) (domain.CompanyReport, error) {
	normalized, ok := domain.NormalizeCNPJ(cnpj)
	if !ok {
		return domain.CompanyReport{}, domain.ErrInvalidCNPJ
	}
	return uc.acts.ReportByEntity(ctx, domain.EntityCNPJ, normalized)
}

// ActStats conta atos por mês.
type ActStats struct{ acts ports.ActRepository }

func NewActStats(a ports.ActRepository) *ActStats { return &ActStats{acts: a} }

func (uc *ActStats) Execute(ctx context.Context, f domain.ActFilter) ([]domain.MonthCount, error) {
	if err := f.Normalize(); err != nil {
		return nil, err
	}
	return uc.acts.CountByMonth(ctx, f)
}
