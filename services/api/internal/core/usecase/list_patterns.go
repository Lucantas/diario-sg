package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type ListPatterns struct{ src ports.PatternSource }

func NewListPatterns(src ports.PatternSource) *ListPatterns { return &ListPatterns{src: src} }

func (uc *ListPatterns) Execute(ctx context.Context) ([]domain.PatternReport, map[string]domain.ActHit, error) {
	catalog := domain.PatternCatalog()
	dispensas, err := uc.src.DispensaActs(ctx)
	if err != nil {
		return nil, nil, err
	}
	counts, err := uc.src.MonthlyActCounts(ctx, []domain.ActType{domain.ActNomeacao, domain.ActExoneracao}, domain.SourceDiarioPrefeitura)
	if err != nil {
		return nil, nil, err
	}
	split := domain.PatternReport{Pattern: catalog[domain.PatternSplitDispensa], Findings: []domain.Finding{}}
	var ids []string
	for _, s := range domain.FindSplitDispensas(dispensas) {
		f := domain.SplitDispensaFinding(s)
		split.Findings = append(split.Findings, f)
		ids = append(ids, f.ActIDs...)
	}
	peaks := domain.PatternReport{Pattern: catalog[domain.PatternElectionHiring], Findings: []domain.Finding{}}
	for _, p := range domain.FindElectionPeaks(counts) {
		peaks.Findings = append(peaks.Findings, domain.ElectionPeakFinding(p))
	}
	acts := map[string]domain.ActHit{}
	if len(ids) > 0 {
		hits, err := uc.src.HitsByIDs(ctx, ids)
		if err != nil {
			return nil, nil, err
		}
		for _, h := range hits {
			acts[h.ID] = h
		}
	}
	return []domain.PatternReport{split, peaks}, acts, nil
}
