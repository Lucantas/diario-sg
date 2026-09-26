package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

var sanctionsHeader = []string{"CADASTRO", "CÓDIGO DA SANÇÃO", "TIPO DE PESSOA", "CPF OU CNPJ DO SANCIONADO", "NOME DO SANCIONADO",
	"RAZÃO SOCIAL - CADASTRO RECEITA", "NÚMERO DO PROCESSO", "CATEGORIA DA SANÇÃO", "DATA INÍCIO SANÇÃO", "DATA FINAL SANÇÃO",
	"DATA PUBLICAÇÃO", "ABRAGÊNCIA DA SANÇÃO", "ÓRGÃO SANCIONADOR", "UF ÓRGÃO SANCIONADOR", "ESFERA ÓRGÃO SANCIONADOR", "FUNDAMENTAÇÃO LEGAL"}

func sanctionRow(register, code, kind, doc string) []string {
	return []string{register, code, kind, doc, "EMPRESA", "EMPRESA LTDA", "1/2025", "Suspensão", "01/01/2025", "01/01/2027",
		"02/01/2025", "No órgão sancionador", "PREFEITURA X", "RJ", "MUNICIPAL", "LEI 14133"}
}

var cepimHeader = []string{"CNPJ ENTIDADE", "NOME ENTIDADE", "NÚMERO CONVÊNIO", "ÓRGÃO CONCEDENTE", "MOTIVO DO IMPEDIMENTO"}

type fakeSanctionSource struct {
	rows   map[string][][]string
	failOn string
}

func (f *fakeSanctionSource) LatestDay(_ context.Context, register string) (time.Time, error) {
	if register == domain.RegisterCNEP {
		return time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), nil
	}
	return time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), nil
}

func (f *fakeSanctionSource) Rows(_ context.Context, register string, _ time.Time, each func(header, row []string) error) (string, error) {
	if register == f.failOn {
		return "", errors.New("status 403")
	}
	header := sanctionsHeader
	if register == domain.RegisterCEPIM {
		header = cepimHeader
	}
	for _, r := range f.rows[register] {
		if err := each(header, r); err != nil {
			return "", err
		}
	}
	return "sha-" + register, nil
}

type fakeSanctionRepo struct {
	saved *domain.SanctionLoad
}

func (r *fakeSanctionRepo) Ready(context.Context) error { return nil }
func (r *fakeSanctionRepo) CitedCNPJs(context.Context) ([]string, error) {
	return []string{"28926250000176"}, nil
}
func (r *fakeSanctionRepo) Save(_ context.Context, load domain.SanctionLoad) error {
	r.saved = &load
	return nil
}

func sanctionsFixture() *fakeSanctionSource {
	return &fakeSanctionSource{rows: map[string][][]string{
		domain.RegisterCEIS: {
			sanctionRow("CEIS", "1", "J", "28926250000176"),
			sanctionRow("CEIS", "2", "J", "28926250000257"),
			sanctionRow("CEIS", "3", "J", "11111111000111"),
			sanctionRow("CEIS", "4", "F", "28926250000"),
			append(sanctionRow("CEIS", "5", "J", "28926250000176")[:10], "quebrada"),
		},
		domain.RegisterCNEP: {sanctionRow("CNEP", "9", "J", "28926250000176")},
		domain.RegisterCEPIM: {
			{"28926250000176", "ASSOCIACAO X", "777", "Ministério Y", "INSTAURACAO DE TOMADA DE CONTAS ESPECIAL"},
			{"11111111000111", "OUTRA", "778", "Ministério Y", "MOTIVO"},
		},
	}}
}

func TestLoadSanctionsKeepsTheCompaniesOfTheCitedBases(t *testing.T) {
	repo, runs := &fakeSanctionRepo{}, &memRuns{}
	uc := NewLoadSanctions(sanctionsFixture(), repo, runs, &memObjects{}, time.Now)

	run, err := uc.Execute(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	var codes []string
	for _, s := range repo.saved.Sanctions {
		codes = append(codes, s.Register+":"+s.Code)
	}
	if strings.Join(codes, ",") != "CEIS:1,CEIS:2,CNEP:9,CEPIM:28926250000176/777" {
		t.Errorf("sanções: %v", codes)
	}
	cnep := repo.saved.Sanctions[2]
	if !cnep.FirstSeen.Equal(time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)) || !cnep.LastSeen.Equal(cnep.FirstSeen) {
		t.Errorf("dia do arquivo: %+v", cnep)
	}
	if run.Source != domain.SourceSanctions || run.Found != 7 || run.Stored != 4 || run.Failed != 1 || run.Skipped != 2 || !run.Valid() {
		t.Errorf("coleta: %+v", run)
	}
	if len(runs.runs) != 1 {
		t.Errorf("coleta não registrada: %+v", runs.runs)
	}
}

func TestLoadSanctionsArchivesOnlyTheCompanyRowsOfTheCitedBases(t *testing.T) {
	raw := &memObjects{}
	uc := NewLoadSanctions(sanctionsFixture(), &fakeSanctionRepo{}, &memRuns{}, raw, time.Now)

	if _, err := uc.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}

	body := gunzip(t, raw.data["raw/cgu_sancoes/2026/09/25/CEIS.csv.gz"])
	if !strings.HasPrefix(body, "CADASTRO;") || strings.Contains(body, "11111111000111") || strings.Contains(body, ";F;") {
		t.Errorf("extração: %q", body)
	}
	var manifest archivedFile
	if err := json.Unmarshal(raw.data["raw/cgu_sancoes/2026/09/24/CNEP.manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SourceSHA256 != "sha-CNEP" || manifest.Rows != 1 {
		t.Errorf("manifesto: %+v", manifest)
	}
}

func TestLoadSanctionsFailureSavesNothing(t *testing.T) {
	src := sanctionsFixture()
	src.failOn = domain.RegisterCNEP
	repo, runs := &fakeSanctionRepo{}, &memRuns{}

	_, err := NewLoadSanctions(src, repo, runs, &memObjects{}, time.Now).Execute(context.Background())

	if err == nil || repo.saved != nil {
		t.Fatalf("falha no CNEP não grava nada: err=%v", err)
	}
	if len(runs.runs) != 1 || !strings.Contains(runs.runs[0].Error, "CNEP") {
		t.Errorf("erro não registrado: %+v", runs.runs)
	}
}
