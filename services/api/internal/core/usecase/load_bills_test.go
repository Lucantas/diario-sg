package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeBillSource struct {
	keys      []domain.BillKey
	pages     map[domain.BillKey]string
	requested []domain.BillKey
}

func (f *fakeBillSource) BaseURL() string { return "https://sicam" }

func (f *fakeBillSource) ProcessKeys(context.Context) ([]domain.BillKey, error) { return f.keys, nil }

func (f *fakeBillSource) ProcessPage(_ context.Context, k domain.BillKey) ([]byte, error) {
	f.requested = append(f.requested, k)
	page, ok := f.pages[k]
	if !ok {
		return nil, errors.New("timeout")
	}
	return []byte(page), nil
}

type fakeBillRepo struct {
	known   map[domain.BillKey]bool
	stale   []domain.BillKey
	batches [][]domain.Bill
}

func (f *fakeBillRepo) Ready(context.Context) error { return nil }

func (f *fakeBillRepo) KnownBills(context.Context) (map[domain.BillKey]bool, error) {
	return f.known, nil
}

func (f *fakeBillRepo) StaleOpenBills(_ context.Context, _ []string, limit int) ([]domain.BillKey, error) {
	if limit < len(f.stale) {
		return f.stale[:limit], nil
	}
	return f.stale, nil
}

func (f *fakeBillRepo) SaveBills(_ context.Context, b []domain.Bill) error {
	f.batches = append(f.batches, b)
	return nil
}

func (f *fakeBillRepo) saved() int {
	n := 0
	for _, b := range f.batches {
		n += len(b)
	}
	return n
}

func billPage(k domain.BillKey) string {
	return fmt.Sprintf(`<html><main><article class="hero-card"><span class="badge"><i class="fas fa-file-alt"></i> PROJETO DE LEI</span>
		<h1 class="hero-title"><span>PROJETO DE LEI Nº 1/%[2]d</span><span>|</span><span>Processo: %[1]d/%[2]d</span></h1></article></main></html>`, k.Number, k.Year)
}

const missingBillPage = `<main><article class="hero-card"><h1 class="hero-title"><span>0 Nº 000/</span><span>|</span><span>Processo: </span></h1></article></main>`

func keysUpTo(n int) []domain.BillKey {
	var out []domain.BillKey
	for i := 1; i <= n; i++ {
		out = append(out, domain.BillKey{Number: i, Year: 2025})
	}
	return out
}

func TestLoadBillsDailyFetchesNewAndStaleUpToMax(t *testing.T) {
	keys := keysUpTo(5)
	src := &fakeBillSource{keys: keys, pages: map[domain.BillKey]string{}}
	for _, k := range keys {
		src.pages[k] = billPage(k)
	}
	repo := &fakeBillRepo{known: map[domain.BillKey]bool{keys[0]: true, keys[1]: true, keys[2]: true}, stale: []domain.BillKey{keys[1], keys[0]}}

	run, err := NewLoadBills(src, repo, &memRuns{}, &memObjects{}, staffNow).Execute(context.Background(), false, 3)

	if err != nil || run.Found != 3 || run.Stored != 3 {
		t.Fatalf("veio %+v %v", run, err)
	}
	want := []domain.BillKey{keys[3], keys[4], keys[1]}
	for i, k := range want {
		if src.requested[i] != k {
			t.Fatalf("ordem: novos primeiro, depois os parados: %v", src.requested)
		}
	}
	if !repo.batches[0][0].FetchedAt.Equal(staffNow()) {
		t.Errorf("lido em: %v", repo.batches[0][0].FetchedAt)
	}
}

func TestLoadBillsFullSavesInBatchesAndArchivesEachBatch(t *testing.T) {
	keys := keysUpTo(5)
	src := &fakeBillSource{keys: keys, pages: map[domain.BillKey]string{}}
	for _, k := range keys {
		src.pages[k] = billPage(k)
	}
	src.pages[keys[2]] = missingBillPage
	repo, raw := &fakeBillRepo{known: map[domain.BillKey]bool{keys[0]: true}}, &memObjects{}
	uc := NewLoadBills(src, repo, &memRuns{}, raw, staffNow)
	uc.batch = 2

	run, err := uc.Execute(context.Background(), true, 0)

	if err != nil || run.Found != 5 || run.Stored != 4 || run.Skipped != 1 || len(repo.batches) != 2 {
		t.Fatalf("veio %+v %v lotes=%d", run, err, len(repo.batches))
	}
	var archived []string
	for _, name := range raw.names {
		if strings.HasSuffix(name, ".jsonl.gz") {
			archived = append(archived, name)
		}
	}
	if len(archived) != 2 || !strings.HasPrefix(archived[0], "raw/sicam_processos/2026/09/26/"+run.ID+"-1") {
		t.Fatalf("bruto: %v", raw.names)
	}
	body := gunzip(t, raw.data[archived[0]])
	if !strings.Contains(body, `"processo":"1-2025"`) || !strings.Contains(body, "<main>") || strings.Contains(body, "<html>") {
		t.Fatalf("linha do bruto: %s", body)
	}
}

func TestLoadBillsReadsAProcessRepeatedInTheSitemapOnlyOnce(t *testing.T) {
	keys := keysUpTo(2)
	src := &fakeBillSource{keys: append(append([]domain.BillKey{}, keys...), keys[0]), pages: map[domain.BillKey]string{}}
	for _, k := range keys {
		src.pages[k] = billPage(k)
	}
	repo := &fakeBillRepo{}

	run, err := NewLoadBills(src, repo, &memRuns{}, &memObjects{}, staffNow).Execute(context.Background(), true, 0)

	if err != nil || run.Found != 2 || len(src.requested) != 2 || repo.saved() != 2 {
		t.Fatalf("processo repetido no sitemap: %+v %v pedidos=%v", run, err, src.requested)
	}
}

func TestLoadBillsKeepsGoingAfterAFailedPageButStopsWhenTheSiteIsDown(t *testing.T) {
	keys := keysUpTo(3)
	src := &fakeBillSource{keys: keys, pages: map[domain.BillKey]string{keys[0]: billPage(keys[0]), keys[2]: billPage(keys[2])}}
	repo := &fakeBillRepo{known: map[domain.BillKey]bool{}}

	run, err := NewLoadBills(src, repo, &memRuns{}, &memObjects{}, staffNow).Execute(context.Background(), true, 0)

	if err != nil || run.Failed != 1 || run.Stored != 2 {
		t.Fatalf("uma página que falha não para a coleta: %+v %v", run, err)
	}

	down := &fakeBillSource{keys: keysUpTo(maxConsecutiveBillFailures + 5), pages: map[domain.BillKey]string{}}
	down.pages[domain.BillKey{Number: 1, Year: 2025}] = billPage(domain.BillKey{Number: 1, Year: 2025})
	repo = &fakeBillRepo{known: map[domain.BillKey]bool{}}
	run, err = NewLoadBills(down, repo, &memRuns{}, &memObjects{}, staffNow).Execute(context.Background(), true, 0)

	if err == nil || run.Failed != maxConsecutiveBillFailures || repo.saved() != 1 {
		t.Fatalf("site fora do ar deveria parar e manter o que já leu: %+v %v salvos=%d", run, err, repo.saved())
	}
}

func TestLoadBillsCountsParserErrorsAsFailed(t *testing.T) {
	keys := keysUpTo(2)
	src := &fakeBillSource{keys: keys, pages: map[domain.BillKey]string{keys[0]: billPage(keys[0]), keys[1]: "<main>sem cabeçalho</main>"}}
	repo := &fakeBillRepo{known: map[domain.BillKey]bool{}}

	run, err := NewLoadBills(src, repo, &memRuns{}, &memObjects{}, staffNow).Execute(context.Background(), true, 0)

	if err != nil || run.Stored != 1 || run.Skipped+run.Failed != 1 {
		t.Fatalf("veio %+v %v", run, err)
	}
}
