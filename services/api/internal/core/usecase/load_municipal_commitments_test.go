package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakePortal struct {
	years map[string]string
	asked []string
}

func (f *fakePortal) Entities(context.Context) ([]byte, error) {
	return []byte(`[{"id_entidade":"1","ds_entidade":"PREFEITURA","ativo":"1"},{"id_entidade":"18","ds_entidade":"FUNDO DE SAUDE","ativo":"1"}]`), nil
}

func (f *fakePortal) Commitments(_ context.Context, year, entity int) ([]byte, error) {
	key := fmt.Sprintf("%d/%d", year, entity)
	f.asked = append(f.asked, key)
	body, ok := f.years[key]
	if !ok {
		return nil, errors.New("fora do ar")
	}
	return []byte(body), nil
}

type fakeMunicipalRepo struct {
	years  []int
	saved  map[int][]domain.MunicipalCommitment
	totals map[int][]domain.MunicipalTotal
}

func (f *fakeMunicipalRepo) Ready(context.Context) error { return nil }

func (f *fakeMunicipalRepo) ReplaceMunicipalYear(_ context.Context, year int, c []domain.MunicipalCommitment, t []domain.MunicipalTotal) error {
	if f.saved == nil {
		f.saved, f.totals = map[int][]domain.MunicipalCommitment{}, map[int][]domain.MunicipalTotal{}
	}
	f.years = append(f.years, year)
	f.saved[year], f.totals[year] = c, t
	return nil
}

const emptyPortalYear = `{"empenhos":[],"totais":{"total_empenhado_acu":"R$ 0,00","total_liquidado_acu":"R$ 0,00","total_pago_acu":"R$ 0,00"}}`

func portalYear(paid string) string {
	return `{"empenhos":[{"nome_razao":"3T","documento_formatado":"38.227.436/0001-90","id_empenho":"7","dt_empenho":"2025-08-05 00:00:00",` +
		`"vl_empenhado_acu":"R$ ` + paid + `","vl_liquidado_acu":"R$ ` + paid + `","vl_pago_acu":"R$ ` + paid + `"}],` +
		`"totais":{"total_empenhado_acu":"R$ ` + paid + `","total_liquidado_acu":"R$ ` + paid + `","total_pago_acu":"R$ ` + paid + `"}}`
}

func TestLoadMunicipalCommitmentsReplacesEachYearWithAllEntities(t *testing.T) {
	src := &fakePortal{years: map[string]string{"2025/1": portalYear("10,00"), "2025/18": emptyPortalYear, "2026/1": portalYear("1,00"), "2026/18": portalYear("2,00")}}
	repo, runs, raw := &fakeMunicipalRepo{}, &memRuns{}, &memObjects{}

	run, err := NewLoadMunicipalCommitments(src, repo, runs, raw, fiscalNow).Execute(context.Background(), 2025, 2026)

	if err == nil {
		t.Fatal("2026 é depois do ano corrente do teste e foi aceito")
	}
	run, err = NewLoadMunicipalCommitments(src, repo, runs, raw, staffNow).Execute(context.Background(), 2025, 2026)
	if err != nil || run.Stored != 3 || fmt.Sprint(repo.years) != "[2025 2026]" {
		t.Fatalf("veio %+v %v %v", run, err, repo.years)
	}
	if len(repo.saved[2025]) != 1 || repo.saved[2025][0].PaidCents != 1000 || len(repo.totals[2025]) != 1 || repo.totals[2025][0].EntityID != 1 {
		t.Errorf("2025: %+v %+v", repo.saved[2025], repo.totals[2025])
	}
	if _, ok := raw.data["raw/pmsg_empenhos/2026/09/26/2026-18.json.gz"]; !ok {
		t.Errorf("bruto: %v", raw.names)
	}
}

func TestLoadMunicipalCommitmentsSkipsARepeatedCommitment(t *testing.T) {
	twice := strings.Replace(portalYear("10,00"), `"empenhos":[{`, `"empenhos":[{"nome_razao":"3T","documento_formatado":"38.227.436/0001-90","id_empenho":"7","dt_empenho":"2025-08-05 00:00:00"},{`, 1)
	src := &fakePortal{years: map[string]string{"2025/1": twice, "2025/18": emptyPortalYear}}
	repo := &fakeMunicipalRepo{}

	run, err := NewLoadMunicipalCommitments(src, repo, &memRuns{}, &memObjects{}, staffNow).Execute(context.Background(), 2025, 2025)

	if err != nil || run.Stored != 1 || run.Skipped != 1 || len(repo.saved[2025]) != 1 {
		t.Fatalf("veio %+v %v %+v", run, err, repo.saved[2025])
	}
}

func TestLoadMunicipalCommitmentsKeepsTheYearWhenAnEntityFails(t *testing.T) {
	src := &fakePortal{years: map[string]string{"2025/1": portalYear("10,00")}}
	repo, runs := &fakeMunicipalRepo{}, &memRuns{}

	_, err := NewLoadMunicipalCommitments(src, repo, runs, &memObjects{}, staffNow).Execute(context.Background(), 2025, 2025)

	if err == nil || len(repo.years) != 0 || runs.runs[0].Error == "" {
		t.Fatalf("veio %v %v %+v", err, repo.years, runs.runs)
	}
}
