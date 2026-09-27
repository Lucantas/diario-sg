package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeRREO struct {
	published map[string]string
	fail      bool
	asked     []string
}

func (f *fakeRREO) RREO(_ context.Context, year, period int) ([]byte, error) {
	key := fmt.Sprintf("%d/%d", year, period)
	f.asked = append(f.asked, key)
	if f.fail {
		return nil, errors.New("fora do ar")
	}
	if body, ok := f.published[key]; ok {
		return []byte(body), nil
	}
	return []byte(`{"items":[]}`), nil
}

func (f *fakeRREO) RREOURL(year, period int) string {
	return fmt.Sprintf("https://siconfi/%d/%d", year, period)
}

type fakeFiscalRepo struct{ saved []domain.FiscalTotal }

func (f *fakeFiscalRepo) Ready(context.Context) error { return nil }

func (f *fakeFiscalRepo) SaveFiscalTotals(_ context.Context, t []domain.FiscalTotal) error {
	f.saved = t
	return nil
}

func rreoBody(paid string) string {
	row := `{"cod_conta":"TotalDespesas","coluna":"%s","valor":%s}`
	return `{"items":[` + fmt.Sprintf(row, "DESPESAS EMPENHADAS ATÉ O BIMESTRE (f)", paid) + "," +
		fmt.Sprintf(row, "DESPESAS LIQUIDADAS ATÉ O BIMESTRE (h)", paid) + "," + fmt.Sprintf(row, "DESPESAS PAGAS ATÉ O BIMESTRE (j)", paid) + `]}`
}

func fiscalNow() time.Time { return time.Date(2018, 9, 27, 10, 0, 0, 0, time.UTC) }

func TestLoadFiscalTotalsTakesTheLatestPublishedPeriodOfEachYear(t *testing.T) {
	src := &fakeRREO{published: map[string]string{"2017/6": rreoBody("100.5"), "2018/4": rreoBody("40")}}
	repo, runs, raw := &fakeFiscalRepo{}, &memRuns{}, &memObjects{}

	run, err := NewLoadFiscalTotals(src, repo, runs, raw, fiscalNow).Execute(context.Background())

	if err != nil || run.Stored != 2 || len(repo.saved) != 2 {
		t.Fatalf("veio %+v %v %+v", run, err, repo.saved)
	}
	if repo.saved[0] != (domain.FiscalTotal{Year: 2017, Period: 6, CommittedCents: 10050, LiquidatedCents: 10050, PaidCents: 10050, SourceURL: "https://siconfi/2017/6"}) ||
		repo.saved[1].Period != 4 {
		t.Errorf("totais: %+v", repo.saved)
	}
	if fmt.Sprint(src.asked) != "[2017/6 2018/6 2018/5 2018/4]" {
		t.Errorf("pedidos: %v", src.asked)
	}
	if _, ok := raw.data["raw/siconfi/2018/09/27/rreo_2018_4.json.gz"]; !ok {
		t.Errorf("bruto: %v", raw.names)
	}
}

func TestLoadFiscalTotalsSavesNothingWhenTheSourceFails(t *testing.T) {
	repo, runs := &fakeFiscalRepo{}, &memRuns{}

	_, err := NewLoadFiscalTotals(&fakeRREO{fail: true}, repo, runs, &memObjects{}, fiscalNow).Execute(context.Background())

	if err == nil || repo.saved != nil || len(runs.runs) != 1 || runs.runs[0].Error == "" {
		t.Fatalf("veio %v %+v %+v", err, repo.saved, runs.runs)
	}
}
