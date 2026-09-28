package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

const (
	defaultBillsPage = 20
	maxBillsPage     = 100
	allBillKinds     = "todos"
)

type FindBills struct {
	bills ports.BillReader
	norms ports.BillNormReader
	now   func() time.Time
}

func NewFindBills(bills ports.BillReader, norms ports.BillNormReader, now func() time.Time) *FindBills {
	return &FindBills{bills: bills, norms: norms, now: now}
}

type BillQuery struct {
	Text, Author, Kind, Status, Phase, Theme, From, To string
	MinIdleDays, Limit, Offset                         int
}

func (uc *FindBills) List(ctx context.Context, q BillQuery) (domain.BillPage, error) {
	f, err := billFilter(q)
	if err != nil {
		return domain.BillPage{}, err
	}
	candidates, err := uc.bills.CandidateBills(ctx, f)
	if err != nil {
		return domain.BillPage{}, err
	}
	today := uc.now()
	page := domain.BillPage{ByPhase: map[domain.BillPhase]int{}, ThemeRule: f.Theme.Rule}
	var kept []domain.BillSummary
	for _, b := range candidates {
		s := domain.SummarizeBill(b, today)
		if s.DaysIdle >= f.MinIdleDays {
			page.ByPhase[s.Phase]++
		}
		if f.Keep(s) {
			kept = append(kept, s)
		}
	}
	if f.MinIdleDays > 0 {
		sort.SliceStable(kept, func(i, j int) bool { return kept[i].DaysIdle > kept[j].DaysIdle })
	}
	page.Total = len(kept)
	page.Items = pageOf(kept, f.Offset, f.Limit)
	cited, err := uc.citedBills(ctx)
	if err != nil {
		return domain.BillPage{}, err
	}
	for i := range page.Items {
		if page.Items[i].Laws, err = uc.laws(ctx, page.Items[i].Bill, cited); err != nil {
			return domain.BillPage{}, err
		}
	}
	return page, nil
}

func pageOf(items []domain.BillSummary, offset, limit int) []domain.BillSummary {
	if offset >= len(items) {
		return []domain.BillSummary{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func billFilter(q BillQuery) (domain.BillFilter, error) {
	f := domain.BillFilter{Text: strings.TrimSpace(q.Text), Author: strings.TrimSpace(q.Author), Status: strings.TrimSpace(q.Status),
		MinIdleDays: q.MinIdleDays, Limit: q.Limit, Offset: q.Offset}
	var err error
	if f.Phase, err = domain.ParseBillPhase(q.Phase); err != nil {
		return f, err
	}
	if f.Theme, err = domain.ParseTheme(q.Theme); err != nil {
		return f, err
	}
	if f.From, err = optionalDay(q.From, "de"); err != nil {
		return f, err
	}
	if f.To, err = optionalDay(q.To, "ate"); err != nil {
		return f, err
	}
	if q.MinIdleDays < 0 || q.Offset < 0 || q.Limit < 0 {
		return f, fmt.Errorf("%w: dias, deslocamento e limite não podem ser negativos", domain.ErrInvalidInput)
	}
	switch kind := strings.ToUpper(strings.TrimSpace(q.Kind)); kind {
	case "":
		f.Kinds = domain.NormativeBillKinds
	case strings.ToUpper(allBillKinds):
		f.Kinds = nil
	default:
		f.Kinds = []string{kind}
	}
	if f.Limit == 0 {
		f.Limit = defaultBillsPage
	}
	if f.Limit > maxBillsPage {
		f.Limit = maxBillsPage
	}
	return f, nil
}

func optionalDay(s, field string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s deve ser AAAA-MM-DD", domain.ErrInvalidInput, field)
	}
	return d, nil
}

func (uc *FindBills) One(ctx context.Context, process string) (domain.BillSummary, error) {
	key, err := domain.ParseBillKey(process)
	if err != nil {
		return domain.BillSummary{}, err
	}
	b, ok, err := uc.bills.BillByKey(ctx, key)
	if err != nil {
		return domain.BillSummary{}, err
	}
	if !ok {
		return domain.BillSummary{}, fmt.Errorf("%w: processo %s", domain.ErrNotFound, key)
	}
	s := domain.SummarizeBill(b, uc.now())
	cited, err := uc.citedBills(ctx)
	if err != nil {
		return domain.BillSummary{}, err
	}
	s.Laws, err = uc.laws(ctx, b, cited)
	return s, err
}

func (uc *FindBills) ForNorm(ctx context.Context, n domain.Norm) (*domain.Bill, domain.Certainty, error) {
	if n.Kind == domain.NormLaw || n.Kind == domain.NormComplementary {
		bills, err := uc.bills.BillsByLaw(ctx, n.Number, n.Year)
		if err != nil || len(bills) > 0 {
			return firstBill(bills), domain.CertaintyExact, err
		}
	}
	ref, ok := domain.BillReference(n.Author)
	if !ok {
		return nil, "", nil
	}
	bills, err := uc.bills.BillsByDoc(ctx, ref)
	if err != nil || len(bills) == 0 {
		return nil, "", err
	}
	return firstBill(bills), domain.CertaintyStrong, nil
}

func firstBill(bills []domain.Bill) *domain.Bill {
	if len(bills) == 0 {
		return nil
	}
	return &bills[0]
}

func (uc *FindBills) citedBills(ctx context.Context) (map[domain.BillDocRef][]domain.Norm, error) {
	norms, err := uc.norms.NormsCitingBills(ctx)
	if err != nil {
		return nil, err
	}
	out := map[domain.BillDocRef][]domain.Norm{}
	for _, n := range norms {
		if ref, ok := domain.BillReference(n.Author); ok {
			out[ref] = append(out[ref], n)
		}
	}
	return out, nil
}

func (uc *FindBills) laws(ctx context.Context, b domain.Bill, cited map[domain.BillDocRef][]domain.Norm) ([]domain.BillLaw, error) {
	if b.LawNumber > 0 {
		kind := domain.NormLaw
		if strings.Contains(b.Kind, "COMPLEMENTAR") {
			kind = domain.NormComplementary
		}
		norms, err := uc.norms.NormsByNumber(ctx, kind, b.LawNumber, b.LawYear)
		if err != nil {
			return nil, err
		}
		norm := domain.Norm{Kind: kind, Number: b.LawNumber, Year: b.LawYear}
		if len(norms) > 0 {
			norm = norms[0]
		}
		return []domain.BillLaw{{Norm: norm, URL: b.LawURL, Certainty: domain.CertaintyExact}}, nil
	}
	var out []domain.BillLaw
	for _, n := range cited[domain.BillDocRef{Kind: b.Kind, Number: b.DocNumber, Year: b.DocYear}] {
		out = append(out, domain.BillLaw{Norm: n, URL: n.TextURL, Certainty: domain.CertaintyStrong})
	}
	return out, nil
}

func (uc *FindBills) ByDoc(ctx context.Context, kind, number string) ([]domain.BillSummary, error) {
	kind = strings.ToUpper(strings.TrimSpace(kind))
	if kind == "" {
		return nil, fmt.Errorf("%w: informe o tipo do documento, como projeto de lei", domain.ErrInvalidInput)
	}
	n, year, _, err := domain.ParseNormNumber(number)
	if err != nil {
		return nil, fmt.Errorf("%w: número do documento %q (use número/ano, como 270/2019)", domain.ErrInvalidInput, number)
	}
	bills, err := uc.bills.BillsByDoc(ctx, domain.BillDocRef{Kind: kind, Number: n, Year: year})
	if err != nil {
		return nil, err
	}
	cited, err := uc.citedBills(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.BillSummary, 0, len(bills))
	for _, b := range bills {
		s := domain.SummarizeBill(b, uc.now())
		if s.Laws, err = uc.laws(ctx, b, cited); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}
