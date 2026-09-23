package usecase

import (
	"context"
	"sort"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

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

type GetCompany struct{ acts ports.ActRepository }

func NewGetCompany(a ports.ActRepository) *GetCompany { return &GetCompany{acts: a} }

func (uc *GetCompany) Execute(ctx context.Context, cnpj string) (domain.CompanyReport, error) {
	normalized, ok := domain.NormalizeCNPJ(cnpj)
	if !ok {
		return domain.CompanyReport{}, domain.ErrInvalidCNPJ
	}
	return uc.acts.ReportByEntity(ctx, domain.EntityCNPJ, normalized)
}

type ActStats struct{ acts ports.ActRepository }

func NewActStats(a ports.ActRepository) *ActStats { return &ActStats{acts: a} }

func (uc *ActStats) Execute(ctx context.Context, f domain.ActFilter) ([]domain.MonthCount, error) {
	if err := f.Normalize(); err != nil {
		return nil, err
	}
	return uc.acts.CountByMonth(ctx, f)
}

type ListOrgans struct{ acts ports.ActRepository }

func NewListOrgans(a ports.ActRepository) *ListOrgans { return &ListOrgans{acts: a} }

func (uc *ListOrgans) Execute(ctx context.Context) ([]domain.OrganCount, error) {
	counts, err := uc.acts.CountByOrgan(ctx)
	if err != nil {
		return nil, err
	}
	merged := make(map[string]int, len(counts))
	for acronym, n := range counts {
		merged[domain.PrincipalOrgan(acronym)] += n
	}
	out := make([]domain.OrganCount, 0, len(merged))
	for acronym, n := range merged {
		out = append(out, domain.OrganCount{Organ: domain.Organ{Acronym: acronym, Name: domain.OrganName(acronym)}, Acts: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Acts != out[j].Acts {
			return out[i].Acts > out[j].Acts
		}
		return out[i].Acronym < out[j].Acronym
	})
	return out, nil
}

type ActFeed struct{ acts ports.ActRepository }

func NewActFeed(a ports.ActRepository) *ActFeed { return &ActFeed{acts: a} }

func (uc *ActFeed) Execute(ctx context.Context, f domain.ActFilter) ([]domain.ActHit, error) {
	if err := f.Normalize(); err != nil {
		return nil, err
	}
	f.Recent, f.Limit, f.Offset = true, domain.FeedLimit, 0
	hits, _, err := uc.acts.Search(ctx, f)
	return hits, err
}
