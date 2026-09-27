package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeNormSource map[string]string

func (f fakeNormSource) Norms(_ context.Context, category string) ([]byte, error) {
	page, ok := f[category]
	if !ok {
		return nil, errors.New("fora do ar")
	}
	return []byte(page), nil
}

func (f fakeNormSource) BaseURL() string { return "https://siapegov/" }

type fakeNormRepo struct{ saved []domain.Norm }

func (f *fakeNormRepo) Ready(context.Context) error { return nil }

func (f *fakeNormRepo) ReplaceNorms(_ context.Context, n []domain.Norm) error {
	f.saved = n
	return nil
}

func normRow(number string) string {
	return `<tr><td>` + number + `</td><td></td><td></td><td></td><td></td><td>ementa</td><td>01/02/2020</td><td></td></tr>`
}

func normFixture() fakeNormSource {
	return fakeNormSource{
		"01": `<table>` + normRow("10/2020") + normRow("10/2020") + normRow("245/202") + `</table>`,
		"03": `<table>` + normRow("1/2021") + `</table>`,
		"02": `<table>` + normRow("1/1990") + `</table>`,
		"05": `<table>` + normRow("10/2020") + `</table>`,
	}
}

func TestLoadNormsReplacesAllKindsAndCountsProblems(t *testing.T) {
	repo, runs, raw := &fakeNormRepo{}, &memRuns{}, &memObjects{}

	run, err := NewLoadNorms(normFixture(), repo, runs, raw, staffNow).Execute(context.Background())

	if err != nil || run.Stored != 4 || run.Found != 6 || run.Skipped != 1 || run.Failed != 1 || len(repo.saved) != 4 {
		t.Fatalf("veio %+v %v %+v", run, err, repo.saved)
	}
	if _, ok := raw.data["raw/siapegov_normas/2026/09/26/decreto.html.gz"]; !ok {
		t.Errorf("bruto: %v", raw.names)
	}
}

func TestLoadNormsSavesNothingWhenACategoryFails(t *testing.T) {
	src := normFixture()
	delete(src, "05")
	repo, runs := &fakeNormRepo{}, &memRuns{}

	_, err := NewLoadNorms(src, repo, runs, &memObjects{}, staffNow).Execute(context.Background())

	if err == nil || repo.saved != nil || runs.runs[0].Error == "" {
		t.Fatalf("veio %v %+v", err, runs.runs)
	}
}
