package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeStaffSource map[int]string

func (f fakeStaffSource) Staff(_ context.Context, year int) ([]byte, error) {
	body, ok := f[year]
	if !ok {
		return nil, fmt.Errorf("ano %d indisponível", year)
	}
	return []byte(body), nil
}

type fakeStaffRepo struct{ years map[int][]domain.StaffRow }

func (f *fakeStaffRepo) Ready(context.Context) error { return nil }

func (f *fakeStaffRepo) ReplaceStaffYear(_ context.Context, year int, rows []domain.StaffRow) error {
	if f.years == nil {
		f.years = map[int][]domain.StaffRow{}
	}
	f.years[year] = rows
	return nil
}

func staffBody(month string) string {
	return `{"SituacoesFuncionais":[{"Anomes":"` + month + `","UnidadeGestora":"PREFEITURA SÃO GONÇALO","Quantidade":10,` +
		`"Remuneracao":1000.5,"SituacaoFuncional":"Efetivo - Estatutário","Grupo":"Efetivo"}]}`
}

var staffNow = func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }

func TestLoadStaffStartsIn2024AndArchivesEachYear(t *testing.T) {
	src := fakeStaffSource{2024: staffBody("2024/01"), 2025: staffBody("2025/01"), 2026: staffBody("2026/01")}
	repo, runs, raw := &fakeStaffRepo{}, &memRuns{}, &memObjects{}

	run, err := NewLoadStaff(src, repo, runs, raw, staffNow).Execute(context.Background(), 2020, 2026)

	if err != nil || run.Stored != 3 || len(repo.years) != 3 || repo.years[2025][0].RemunerationCents != 100050 {
		t.Fatalf("veio %+v %v %+v", run, err, repo.years)
	}
	if _, ok := raw.data["raw/tce_pessoal/2026/09/26/2024.json.gz"]; !ok {
		t.Fatalf("bruto: %v", raw.names)
	}
	if _, ok := raw.data["raw/tce_pessoal/2026/09/26/2024.manifest.json"]; !ok || len(runs.runs) != 1 {
		t.Fatalf("manifesto e execução: %v %+v", raw.names, runs.runs)
	}
}

func TestLoadStaffKeepsAYearThatCameEmpty(t *testing.T) {
	src := fakeStaffSource{2025: `{"SituacoesFuncionais":[]}`, 2026: staffBody("2026/01")}
	repo := &fakeStaffRepo{}

	run, err := NewLoadStaff(src, repo, &memRuns{}, &memObjects{}, staffNow).Execute(context.Background(), 0, 0)

	if err != nil || run.Stored != 1 {
		t.Fatalf("veio %+v %v", run, err)
	}
	if _, replaced := repo.years[2025]; replaced {
		t.Fatal("ano vazio não deveria trocar o que existe")
	}
}

func TestLoadStaffDoesNothingBefore2024(t *testing.T) {
	runs := &memRuns{}

	run, err := NewLoadStaff(fakeStaffSource{}, &fakeStaffRepo{}, runs, &memObjects{}, staffNow).Execute(context.Background(), 2020, 2023)

	if err != nil || run.ID != "" || len(runs.runs) != 0 {
		t.Fatalf("veio %+v %v %+v", run, err, runs.runs)
	}
}

func TestLoadStaffStopsOnASourceError(t *testing.T) {
	runs := &memRuns{}

	_, err := NewLoadStaff(fakeStaffSource{}, &fakeStaffRepo{}, runs, &memObjects{}, staffNow).Execute(context.Background(), 2025, 2026)

	if err == nil || len(runs.runs) != 1 || runs.runs[0].Error == "" {
		t.Fatalf("veio %v %+v", err, runs.runs)
	}
}
