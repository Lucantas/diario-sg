package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakePatternSource struct {
	dispensas   []domain.DispensaAct
	addenda     []domain.AddendumAct
	emergencies []domain.EmergencyAct
	counts      []domain.MonthlyActCount
	askedIDs    []string
}

func (f *fakePatternSource) DispensaActs(context.Context) ([]domain.DispensaAct, error) {
	return f.dispensas, nil
}
func (f *fakePatternSource) AddendumActs(context.Context) ([]domain.AddendumAct, error) {
	return f.addenda, nil
}
func (f *fakePatternSource) EmergencyActs(context.Context) ([]domain.EmergencyAct, error) {
	return f.emergencies, nil
}
func (f *fakePatternSource) MonthlyActCounts(context.Context, []domain.ActType, string) ([]domain.MonthlyActCount, error) {
	return f.counts, nil
}
func (f *fakePatternSource) HitsByIDs(_ context.Context, ids []string) ([]domain.ActHit, error) {
	f.askedIDs = ids
	var out []domain.ActHit
	for _, id := range ids {
		out = append(out, domain.ActHit{Act: domain.Act{ID: id}})
	}
	return out, nil
}

type fakeSupplierSource struct {
	acts      []domain.PanelAct
	profiles  map[string]domain.SupplierProfile
	sanctions []domain.Sanction
}

func (f *fakeSupplierSource) PanelActs(context.Context, string) ([]domain.PanelAct, error) {
	return f.acts, nil
}
func (f *fakeSupplierSource) SupplierProfiles(context.Context) (map[string]domain.SupplierProfile, error) {
	return f.profiles, nil
}
func (f *fakeSupplierSource) AllSanctions(context.Context) ([]domain.Sanction, error) {
	return f.sanctions, nil
}

func TestListPatternsAddsTheSupplierPatternsWithTheirActs(t *testing.T) {
	day := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	openedAt := day.AddDate(0, -1, 0)
	suppliers := &fakeSupplierSource{
		acts: []domain.PanelAct{{ActID: "contrato", CNPJ: "11222333000181", PublishedAt: day, ValueCents: 50_000_000, Type: domain.ActContrato,
			Title: "EXTRATO DO CONTRATO 1/SEMED/2025", Head: "EXTRATO DO CONTRATO 1/SEMED/2025", Refs: []string{"contrato:1/SEMED/2025"}}},
		profiles: map[string]domain.SupplierProfile{"11222333000181": {CNPJ: "11222333000181", Name: "EMPRESA NOVA LTDA", Headquarters: true,
			OpenedAt: &openedAt, CapitalCents: 100_000, LegalNature: "Sociedade Empresária Limitada"}},
	}
	src := &fakePatternSource{}

	reports, acts, err := NewListPatterns(src, suppliers).Execute(context.Background())

	if err != nil || len(reports) != 9 {
		t.Fatalf("veio %+v %v", reports, err)
	}
	want := []domain.PatternID{domain.PatternNewCompany, domain.PatternUndercapitalized, domain.PatternSharedPartner, domain.PatternSharedAddress, domain.PatternSanctioned}
	for i, id := range want {
		if reports[4+i].Pattern.ID != id || reports[4+i].Pattern.Rule == "" {
			t.Fatalf("padrão %d: %+v", 4+i, reports[4+i].Pattern)
		}
	}
	if len(reports[4].Findings) != 1 || len(reports[5].Findings) != 1 || acts["contrato"].ID != "contrato" {
		t.Fatalf("achados: %+v %+v %v", reports[4].Findings, reports[5].Findings, acts)
	}
}

func TestListPatternsReturnsEveryPatternWithItsFindingsAndActs(t *testing.T) {
	body := "art. 75, inciso II, da Lei 14.133"
	day := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	src := &fakePatternSource{dispensas: []domain.DispensaAct{
		{ActID: "a", CNPJ: "11222333000181", PublishedAt: day, ValueCents: 4000000, Body: body, Processes: []domain.DispensaProcess{{Key: "1", Label: "1"}}},
		{ActID: "b", CNPJ: "11222333000181", PublishedAt: day.AddDate(0, 1, 0), ValueCents: 4000000, Body: body, Processes: []domain.DispensaProcess{{Key: "2", Label: "2"}}},
	}}

	src.addenda = []domain.AddendumAct{{ActID: "aditivo", ContractKeys: []string{"7/2015"}, Organ: "FMS", PublishedAt: day, Title: "TERMO ADITIVO", IncreaseBP: 4637}}

	reports, acts, err := NewListPatterns(src, &fakeSupplierSource{}).Execute(context.Background())

	if err != nil || len(reports) != 9 {
		t.Fatalf("veio %+v %v", reports, err)
	}
	if reports[0].Pattern.ID != domain.PatternSplitDispensa || len(reports[0].Findings) != 1 || reports[1].Pattern.ID != domain.PatternExcessiveAddenda || len(reports[1].Findings) != 1 ||
		reports[2].Pattern.ID != domain.PatternRenewedEmergency || reports[3].Pattern.ID != domain.PatternElectionHiring {
		t.Fatalf("relatórios inesperados: %+v", reports)
	}
	if len(acts) != 3 || acts["a"].ID != "a" || acts["aditivo"].ID != "aditivo" || len(src.askedIDs) != 3 {
		t.Fatalf("atos inesperados: %+v %v", acts, src.askedIDs)
	}
}
