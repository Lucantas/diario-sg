package usecase

import (
	"context"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type ListPatterns struct {
	src       ports.PatternSource
	suppliers ports.SupplierPatternSource
}

func NewListPatterns(src ports.PatternSource, suppliers ports.SupplierPatternSource) *ListPatterns {
	return &ListPatterns{src: src, suppliers: suppliers}
}

func (uc *ListPatterns) Execute(ctx context.Context) ([]domain.PatternReport, map[string]domain.ActHit, error) {
	catalog := domain.PatternCatalog()
	dispensas, err := uc.src.DispensaActs(ctx)
	if err != nil {
		return nil, nil, err
	}
	addenda, err := uc.src.AddendumActs(ctx)
	if err != nil {
		return nil, nil, err
	}
	emergencies, err := uc.src.EmergencyActs(ctx)
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
	excessive := domain.PatternReport{Pattern: catalog[domain.PatternExcessiveAddenda], Findings: []domain.Finding{}}
	for _, e := range domain.FindExcessiveAddenda(addenda) {
		f := domain.ExcessiveAddendumFinding(e)
		excessive.Findings = append(excessive.Findings, f)
		ids = append(ids, f.ActIDs...)
	}
	renewed := domain.PatternReport{Pattern: catalog[domain.PatternRenewedEmergency], Findings: []domain.Finding{}}
	for _, r := range domain.FindRenewedEmergencies(emergencies) {
		f := domain.RenewedEmergencyFinding(r)
		renewed.Findings = append(renewed.Findings, f)
		ids = append(ids, f.ActIDs...)
	}
	peaks := domain.PatternReport{Pattern: catalog[domain.PatternElectionHiring], Findings: []domain.Finding{}}
	for _, p := range domain.FindElectionPeaks(counts) {
		peaks.Findings = append(peaks.Findings, domain.ElectionPeakFinding(p))
	}
	supplierReports, err := uc.supplierReports(ctx, catalog)
	if err != nil {
		return nil, nil, err
	}
	for _, rep := range supplierReports {
		for _, f := range rep.Findings {
			ids = append(ids, f.ActIDs...)
		}
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
	return append([]domain.PatternReport{split, excessive, renewed, peaks}, supplierReports...), acts, nil
}

func (uc *ListPatterns) supplierReports(ctx context.Context, catalog map[domain.PatternID]domain.Pattern) ([]domain.PatternReport, error) {
	acts, err := uc.suppliers.PanelActs(ctx, domain.SourceDiarioPrefeitura)
	if err != nil {
		return nil, err
	}
	profiles, err := uc.suppliers.SupplierProfiles(ctx)
	if err != nil {
		return nil, err
	}
	sanctions, err := uc.suppliers.AllSanctions(ctx)
	if err != nil {
		return nil, err
	}
	contracts := domain.SupplierContracts(acts)
	payment, err := uc.paymentReports(ctx, catalog, acts, contracts, profiles)
	if err != nil {
		return nil, err
	}
	pncp, err := uc.pncpReport(ctx, catalog, profiles)
	if err != nil {
		return nil, err
	}
	return append([]domain.PatternReport{
		reportOf(catalog[domain.PatternNewCompany], domain.FindNewCompanyContracts(contracts, profiles), domain.NewCompanyFinding),
		reportOf(catalog[domain.PatternUndercapitalized], domain.FindUndercapitalizedContracts(contracts, profiles), domain.UndercapitalizedFinding),
		reportOf(catalog[domain.PatternSharedPartner], domain.FindSharedPartners(contracts, profiles), domain.SharedPartnerFinding),
		reportOf(catalog[domain.PatternSharedAddress], domain.FindSharedAddresses(contracts, profiles), domain.SharedAddressFinding),
		reportOf(catalog[domain.PatternSanctioned], domain.FindSanctionedContracts(contracts, profiles, sanctions), domain.SanctionedFinding),
	}, append(payment, pncp)...), nil
}

func (uc *ListPatterns) paymentReports(ctx context.Context, catalog map[domain.PatternID]domain.Pattern, acts []domain.PanelAct,
	contracts []domain.SupplierContract, profiles map[string]domain.SupplierProfile) ([]domain.PatternReport, error) {
	paid, err := uc.suppliers.PaidCreditors(ctx)
	if err != nil {
		return nil, err
	}
	cited, err := uc.suppliers.CitedInDiario(ctx)
	if err != nil {
		return nil, err
	}
	coverage, err := uc.suppliers.PaymentsCoverage(ctx)
	if err != nil {
		return nil, err
	}
	unnamed, err := uc.suppliers.PanelActsWithoutCNPJ(ctx, domain.SourceDiarioPrefeitura)
	if err != nil {
		return nil, err
	}
	announced := append(append([]domain.PanelAct{}, acts...), domain.AttributeByName(unnamed, paid, profiles)...)
	return []domain.PatternReport{
		reportOf(catalog[domain.PatternPaidUnpublished], domain.FindPaidWithoutPublication(paid, cited, profiles), domain.PaidWithoutPublicationFinding),
		reportOf(catalog[domain.PatternUnpaidContract], domain.FindUnpaidContracts(contracts, paid, coverage, profiles), domain.UnpaidContractFinding),
		reportOf(catalog[domain.PatternPaidAbove], domain.FindPaidAboveAnnounced(domain.SupplierContracts(announced),
			domain.AmendedBySupplier(announced), paid, profiles), domain.PaidAboveAnnouncedFinding),
	}, nil
}

func (uc *ListPatterns) pncpReport(ctx context.Context, catalog map[domain.PatternID]domain.Pattern, profiles map[string]domain.SupplierProfile) (domain.PatternReport, error) {
	contracts, err := uc.suppliers.AllPNCPContracts(ctx)
	if err != nil {
		return domain.PatternReport{}, err
	}
	cited, err := uc.suppliers.CitedInDiario(ctx)
	if err != nil {
		return domain.PatternReport{}, err
	}
	processes, err := uc.suppliers.CitedProcesses(ctx)
	if err != nil {
		return domain.PatternReport{}, err
	}
	last, err := uc.suppliers.LatestGazetteDay(ctx, domain.SourceDiarioPrefeitura)
	if err != nil {
		return domain.PatternReport{}, err
	}
	var unnamed []domain.PanelAct
	for _, source := range []string{domain.SourceDiarioPrefeitura, domain.SourceDiarioCamara} {
		acts, err := uc.suppliers.PanelActsWithoutCNPJ(ctx, source)
		if err != nil {
			return domain.PatternReport{}, err
		}
		unnamed = append(unnamed, acts...)
	}
	citations := domain.PNCPCitations{CNPJs: cited, Processes: processes, Names: domain.PNCPCitedNames(unnamed)}
	return reportOf(catalog[domain.PatternPNCPWithoutExtract], domain.FindPNCPWithoutExtract(contracts, citations, last, profiles),
		domain.PNCPWithoutExtractFinding), nil
}

func reportOf[T any](p domain.Pattern, items []T, finding func(T) domain.Finding) domain.PatternReport {
	rep := domain.PatternReport{Pattern: p, Findings: []domain.Finding{}}
	for _, item := range items {
		rep.Findings = append(rep.Findings, finding(item))
	}
	return rep
}
