package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeOversightSource map[string]string

func (f fakeOversightSource) Dataset(_ context.Context, name string) ([]byte, error) {
	body, ok := f[name]
	if !ok {
		return nil, fmt.Errorf("%s indisponível", name)
	}
	return []byte(body), nil
}

type fakeOversightRepo struct {
	saved *domain.TCEOversight
}

func (f *fakeOversightRepo) Ready(context.Context) error { return nil }

func (f *fakeOversightRepo) ReplaceOversight(_ context.Context, o domain.TCEOversight) error {
	f.saved = &o
	return nil
}

func oversightFixture() fakeOversightSource {
	return fakeOversightSource{
		datasetAccounts:  `[{"Municipio":"SÃO GONÇALO","Ano":2024,"Indicador":"FAVORÁVEL","Processo":"213638-8/2025","Responsavel":"PREFEITO"}]`,
		datasetPenalties: `[{"Processo":"200473-0/2015","AnoCondenacao":2022,"ValorPenalidade":1,"Condenacao":"657/2022-2","Ente":"SAO GONCALO","DataSessao":"2022-05-25T00:00:00"}]`,
		datasetWorks:     `{"Obras":[{"Ente":"SAO GONCALO","NumeroContrato":"1","CNPJContratada":"01992029000160","DataParalisacao":"2016-08-01"}]}`,
	}
}

func TestLoadOversightReplacesTheThreeSetsAndArchivesThem(t *testing.T) {
	repo, runs, raw := &fakeOversightRepo{}, &memRuns{}, &memObjects{}

	run, err := NewLoadOversight(oversightFixture(), repo, runs, raw, staffNow).Execute(context.Background())

	if err != nil || run.Stored != 3 || repo.saved == nil || len(repo.saved.Works) != 1 || repo.saved.Works[0].CNPJ != "01992029000160" {
		t.Fatalf("veio %+v %v %+v", run, err, repo.saved)
	}
	for _, name := range []string{datasetAccounts, datasetPenalties, datasetWorks} {
		if _, ok := raw.data["raw/tce_controle/2026/09/26/"+name+".json.gz"]; !ok {
			t.Errorf("bruto de %s: %v", name, raw.names)
		}
		if _, ok := raw.data["raw/tce_controle/2026/09/26/"+name+".manifest.json"]; !ok {
			t.Errorf("manifesto de %s", name)
		}
	}
}

func TestLoadOversightSavesNothingWhenASetFails(t *testing.T) {
	src := oversightFixture()
	delete(src, datasetWorks)
	repo, runs := &fakeOversightRepo{}, &memRuns{}

	_, err := NewLoadOversight(src, repo, runs, &memObjects{}, staffNow).Execute(context.Background())

	if err == nil || repo.saved != nil || len(runs.runs) != 1 || runs.runs[0].Error == "" {
		t.Fatalf("veio %v %+v %+v", err, repo.saved, runs.runs)
	}
}
