package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeBillReader struct {
	bills  []domain.Bill
	filter domain.BillFilter
}

func (f *fakeBillReader) CandidateBills(_ context.Context, filter domain.BillFilter) ([]domain.Bill, error) {
	f.filter = filter
	return f.bills, nil
}

func (f *fakeBillReader) BillByKey(_ context.Context, k domain.BillKey) (domain.Bill, bool, error) {
	for _, b := range f.bills {
		if b.Key == k {
			return b, true, nil
		}
	}
	return domain.Bill{}, false, nil
}

func (f *fakeBillReader) BillsByDoc(_ context.Context, ref domain.BillDocRef) ([]domain.Bill, error) {
	var out []domain.Bill
	for _, b := range f.bills {
		if b.Kind == ref.Kind && b.DocNumber == ref.Number && b.DocYear == ref.Year {
			out = append(out, b)
		}
	}
	return out, nil
}

func (f *fakeBillReader) BillsByLaw(_ context.Context, kind domain.NormKind, number, year int) ([]domain.Bill, error) {
	var out []domain.Bill
	for _, b := range f.bills {
		if b.LawKind == kind && b.LawNumber == number && b.LawYear == year {
			out = append(out, b)
		}
	}
	return out, nil
}

type fakeBillNorms []domain.Norm

func (f fakeBillNorms) NormsByNumber(_ context.Context, kind domain.NormKind, number, year int) ([]domain.Norm, error) {
	var out []domain.Norm
	for _, n := range f {
		if n.Kind == kind && n.Number == number && n.Year == year {
			out = append(out, n)
		}
	}
	return out, nil
}

func (f fakeBillNorms) NormsCitingBills(context.Context) ([]domain.Norm, error) { return f, nil }

var billsToday = func() time.Time { return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) }

func idleBill(number int, daysAgo int, text string) domain.Bill {
	return domain.Bill{Key: domain.BillKey{Number: number, Year: 2025}, Kind: "PROJETO DE LEI", DocNumber: number, DocYear: 2025,
		Events: []domain.BillEvent{{Position: 1, At: billsToday().AddDate(0, 0, -daysAgo), Text: text}}}
}

func TestFindBillsListFiltersPhaseSortsByIdleAndPages(t *testing.T) {
	reader := &fakeBillReader{bills: []domain.Bill{
		idleBill(1, 10, "Recebido na Comissão"),
		idleBill(2, 400, "Recebido na Comissão"),
		idleBill(3, 200, "Recebido na Comissão"),
		idleBill(4, 500, "Processo Arquivado"),
	}}
	uc := NewFindBills(reader, fakeBillNorms{}, billsToday)

	page, err := uc.List(context.Background(), BillQuery{Phase: "em_comissao", MinIdleDays: 100, Theme: "meio_ambiente", Limit: 1, Offset: 1})

	if err != nil || page.Total != 2 || len(page.Items) != 1 || page.Items[0].Bill.Key.Number != 3 || page.Items[0].DaysIdle != 200 {
		t.Fatalf("veio %+v %v", page, err)
	}
	if page.ByPhase[domain.PhaseCommittee] != 2 || page.ByPhase[domain.PhaseArchived] != 1 || page.ThemeRule == "" {
		t.Fatalf("contagem por fase conta os outros filtros, não a fase: %+v", page.ByPhase)
	}
	if len(reader.filter.Kinds) != len(domain.NormativeBillKinds) || reader.filter.Theme.Slug != "meio_ambiente" {
		t.Fatalf("filtro: %+v", reader.filter)
	}
}

func TestFindBillsAllKindsAndValidation(t *testing.T) {
	reader := &fakeBillReader{}
	uc := NewFindBills(reader, fakeBillNorms{}, billsToday)

	if _, err := uc.List(context.Background(), BillQuery{Kind: "todos"}); err != nil || reader.filter.Kinds != nil {
		t.Fatalf("todos: %v %v", reader.filter.Kinds, err)
	}
	if _, err := uc.List(context.Background(), BillQuery{Kind: "projeto de lei"}); err != nil || len(reader.filter.Kinds) != 1 || reader.filter.Kinds[0] != "PROJETO DE LEI" {
		t.Fatalf("um tipo: %v %v", reader.filter.Kinds, err)
	}
	for _, q := range []BillQuery{{Phase: "sancionado"}, {Theme: "saude"}, {From: "ontem"}, {MinIdleDays: -1}} {
		if _, err := uc.List(context.Background(), q); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("%+v: %v", q, err)
		}
	}
	if _, err := uc.One(context.Background(), "abc"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("processo inválido: %v", err)
	}
	if _, err := uc.One(context.Background(), "9/2025"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("processo inexistente: %v", err)
	}
}

func TestFindBillsLinksLawsByBadgeAndByNormAuthor(t *testing.T) {
	badge := idleBill(1, 10, "Lei nº. 1147/2020 de 05/02/2020")
	badge.LawKind, badge.LawNumber, badge.LawYear, badge.LawURL = domain.NormLaw, 1147, 2020, "https://sicam/lei/216"
	cited := idleBill(133, 10, "Processo Arquivado")
	cited.DocYear = 2019
	norms := fakeBillNorms{
		{Kind: domain.NormLaw, Number: 1147, Year: 2020, Summary: "CAPOEIRA"},
		{Kind: domain.NormLaw, Number: 1131, Year: 2020, Author: "PROJETO DE LEI Nº 0133/2019 VEREADOR X", Summary: "ANIMAIS"},
	}
	uc := NewFindBills(&fakeBillReader{bills: []domain.Bill{badge, cited}}, norms, billsToday)

	one, err := uc.One(context.Background(), "1/2025")
	if err != nil || len(one.Laws) != 1 || one.Laws[0].Norm.Number != 1147 || one.Laws[0].Certainty != domain.CertaintyExact || one.Laws[0].URL != "https://sicam/lei/216" {
		t.Fatalf("pelo selo: %+v %v", one.Laws, err)
	}
	other, err := uc.One(context.Background(), "133/2025")
	if err != nil || len(other.Laws) != 1 || other.Laws[0].Norm.Number != 1131 || other.Laws[0].Certainty != domain.CertaintyStrong {
		t.Fatalf("pelo autor da norma: %+v %v", other.Laws, err)
	}

	bill, certainty, err := uc.ForNorm(context.Background(), norms[1])
	if err != nil || bill == nil || bill.Key.Number != 133 || certainty != domain.CertaintyStrong {
		t.Fatalf("norma → projeto: %+v %v %v", bill, certainty, err)
	}
	bill, certainty, err = uc.ForNorm(context.Background(), norms[0])
	if err != nil || bill == nil || bill.Key.Number != 1 || certainty != domain.CertaintyExact {
		t.Fatalf("norma → projeto pelo selo: %+v %v %v", bill, certainty, err)
	}
}

func TestFindBillsLinksAResolutionBadgeWithoutMistakingItForALaw(t *testing.T) {
	resolution := idleBill(3997, 10, "Resolução 1054/2026 de 19/08/2026")
	resolution.Kind = "PROJETO DE RESOLUÇÃO"
	resolution.LawKind, resolution.LawNumber, resolution.LawYear, resolution.LawURL = domain.NormResolution, 1054, 2026, "https://sicam/lei/871"
	sameNumberLaw := domain.Norm{Kind: domain.NormLaw, Number: 1054, Year: 2026, Summary: "OUTRA COISA"}
	uc := NewFindBills(&fakeBillReader{bills: []domain.Bill{resolution}}, fakeBillNorms{sameNumberLaw}, billsToday)

	one, err := uc.One(context.Background(), "3997/2025")
	if err != nil || len(one.Laws) != 1 || one.Laws[0].Norm.Kind != domain.NormResolution || one.Laws[0].Norm.Summary != "" || one.Phase != domain.PhaseLaw {
		t.Fatalf("resolução: %+v %v", one, err)
	}
	bill, _, err := uc.ForNorm(context.Background(), sameNumberLaw)
	if err != nil || bill != nil {
		t.Fatalf("a lei de mesmo número não é a resolução: %+v %v", bill, err)
	}
}

func TestFindBillsByDocument(t *testing.T) {
	b := idleBill(3865, 10, "Processo Arquivado")
	b.DocNumber, b.DocYear = 270, 2019
	uc := NewFindBills(&fakeBillReader{bills: []domain.Bill{b}}, fakeBillNorms{}, billsToday)

	got, err := uc.ByDoc(context.Background(), "projeto de lei", "270/2019")
	if err != nil || len(got) != 1 || got[0].Bill.Key.Number != 3865 || got[0].Phase != domain.PhaseArchived {
		t.Fatalf("por documento: %+v %v", got, err)
	}
	if _, err := uc.ByDoc(context.Background(), "", "270/2019"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("sem tipo: %v", err)
	}
	if _, err := uc.ByDoc(context.Background(), "projeto de lei", "abc"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("número inválido: %v", err)
	}
}
