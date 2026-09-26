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

type fakeRegistrySource struct {
	files  map[string][][]string
	failOn string
}

func (f *fakeRegistrySource) LatestMonth(context.Context) (string, error) { return "2026-09", nil }
func (f *fakeRegistrySource) Files(context.Context, string) ([]string, error) {
	return []string{"Cnaes.zip", "Empresas0.zip", "Estabelecimentos0.zip", "Simples.zip", "Socios0.zip"}, nil
}
func (f *fakeRegistrySource) Rows(_ context.Context, _, file string, each func([]string) error) (string, error) {
	if file == f.failOn {
		return "", errors.New("timeout")
	}
	for _, row := range f.files[file] {
		if err := each(row); err != nil {
			return "", err
		}
	}
	return "sha-" + file, nil
}
func (f *fakeRegistrySource) Codes(context.Context, string) (domain.RegistryCodes, error) {
	return domain.RegistryCodes{Natures: map[string]string{"2062": "Sociedade Empresária Limitada"}}, nil
}

type fakeRegistryRepo struct {
	notReady error
	cited    []string
	month    time.Time
	load     *domain.RegistryLoad
	replaced int
}

func (r *fakeRegistryRepo) Ready(context.Context) error                  { return r.notReady }
func (r *fakeRegistryRepo) CitedCNPJs(context.Context) ([]string, error) { return r.cited, nil }
func (r *fakeRegistryRepo) Replace(_ context.Context, month time.Time, load domain.RegistryLoad) error {
	r.month, r.load = month, &load
	r.replaced++
	return nil
}

type memRuns struct{ runs []domain.FetchRun }

func (m *memRuns) Save(_ context.Context, r domain.FetchRun) error {
	m.runs = append(m.runs, r)
	return nil
}

func establishmentRow(base, order, dv string) []string {
	row := make([]string, 30)
	copy(row, []string{base, order, dv, "1", "", "02", "20050518", "00", "", "", "20050518", "4120400", ""})
	return row
}

func registryFixture() *fakeRegistrySource {
	return &fakeRegistrySource{files: map[string][][]string{
		"Empresas0.zip": {
			{"07396865", "CONSTRUTORA EXEMPLO LTDA", "2062", "49", "1500000,00", "03", ""},
			{"99999999", "NÃO CITADA LTDA", "2062", "49", "0,00", "01", ""},
		},
		"Estabelecimentos0.zip": {
			establishmentRow("07396865", "0001", "04"),
			establishmentRow("07396865", "0002", "95"),
			establishmentRow("99999999", "0001", "00"),
		},
		"Socios0.zip": {
			{"07396865", "2", "FULANO DE TAL", "***123456**", "49", "20050518", "", "", "", "00", "6"},
			{"99999999", "2", "OUTRO", "***000000**", "49", "20050518", "", "", "", "00", "6"},
			{"07396865", "9", "LINHA QUEBRADA", "", "49", "", "", "", "", "", ""},
		},
		"Simples.zip": {{"07396865", "S"}},
	}}
}

func TestLoadRegistryKeepsOnlyTheCitedCompanies(t *testing.T) {
	repo := &fakeRegistryRepo{cited: []string{"07396865000104", "11111111000111"}}
	runs, raw := &memRuns{}, &memObjects{}
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	uc := NewLoadRegistry(registryFixture(), repo, runs, raw, func() time.Time { return now })

	run, err := uc.Execute(context.Background(), "")

	if err != nil {
		t.Fatal(err)
	}
	if repo.replaced != 1 || !repo.month.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("troca: %d vezes, mês %v", repo.replaced, repo.month)
	}
	if len(repo.load.Companies) != 1 || repo.load.Companies[0].LegalNature != "Sociedade Empresária Limitada" {
		t.Errorf("empresas: %+v", repo.load.Companies)
	}
	if len(repo.load.Establishments) != 1 || repo.load.Establishments[0].CNPJ != "07396865000104" {
		t.Errorf("só o estabelecimento citado: %+v", repo.load.Establishments)
	}
	if len(repo.load.Partners) != 1 || repo.load.Partners[0].Name != "FULANO DE TAL" {
		t.Errorf("sócios: %+v", repo.load.Partners)
	}
	if run.Source != domain.SourceReceita || run.Found != 2 || run.Stored != 1 || run.Skipped != 1 || run.Failed != 1 || !run.Valid() {
		t.Errorf("coleta: %+v", run)
	}
	if len(runs.runs) != 1 {
		t.Errorf("a coleta deveria ser registrada: %+v", runs.runs)
	}
}

func TestLoadRegistryArchivesTheFilteredRowsWithAManifest(t *testing.T) {
	raw := &memObjects{}
	uc := NewLoadRegistry(registryFixture(), &fakeRegistryRepo{cited: []string{"07396865000104"}}, &memRuns{}, raw, time.Now)

	if _, err := uc.Execute(context.Background(), "2026-09"); err != nil {
		t.Fatal(err)
	}

	gz, ok := raw.data["raw/receita_cnpj/2026/09/01/Empresas0.csv.gz"]
	if !ok {
		t.Fatalf("extração da empresa não arquivada: %v", keys(raw.data))
	}
	body := gunzip(t, gz)
	if !strings.Contains(body, "CONSTRUTORA EXEMPLO LTDA") || strings.Contains(body, "NÃO CITADA") {
		t.Errorf("extração: %q", body)
	}
	var manifest map[string]struct {
		SourceSHA256 string `json:"source_sha256"`
		Rows         int    `json:"rows"`
	}
	if err := json.Unmarshal(raw.data["raw/receita_cnpj/2026/09/01/manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["Empresas0.zip"].SourceSHA256 != "sha-Empresas0.zip" || manifest["Empresas0.zip"].Rows != 1 {
		t.Errorf("manifesto: %+v", manifest)
	}
	if _, ok := raw.data["raw/receita_cnpj/2026/09/01/Simples.csv.gz"]; ok {
		t.Error("Simples não é carregado")
	}
}

func TestLoadRegistryFailureReplacesNothingAndRecordsTheError(t *testing.T) {
	src := registryFixture()
	src.failOn = "Socios0.zip"
	repo, runs := &fakeRegistryRepo{cited: []string{"07396865000104"}}, &memRuns{}
	uc := NewLoadRegistry(src, repo, runs, &memObjects{}, time.Now)

	_, err := uc.Execute(context.Background(), "2026-09")

	if err == nil || repo.replaced != 0 {
		t.Fatalf("falha num arquivo não troca nada: err=%v trocas=%d", err, repo.replaced)
	}
	if len(runs.runs) != 1 || !strings.Contains(runs.runs[0].Error, "Socios0.zip") {
		t.Errorf("erro não registrado: %+v", runs.runs)
	}
}

func TestLoadRegistryStopsBeforeReadingWhenTheTablesAreMissing(t *testing.T) {
	src := registryFixture()
	src.failOn = "Empresas0.zip"
	repo, runs := &fakeRegistryRepo{notReady: errors.New(`relation "rf_partners" does not exist`)}, &memRuns{}
	uc := NewLoadRegistry(src, repo, runs, &memObjects{}, time.Now)

	_, err := uc.Execute(context.Background(), "2026-09")

	if err == nil || !strings.Contains(err.Error(), "rf_partners") || strings.Contains(err.Error(), "timeout") {
		t.Fatalf("deveria parar antes de ler os arquivos: %v", err)
	}
	if len(runs.runs) != 1 || runs.runs[0].Error == "" {
		t.Errorf("a falha deveria ser registrada: %+v", runs.runs)
	}
}

func TestLoadRegistryRejectsAMalformedMonth(t *testing.T) {
	uc := NewLoadRegistry(registryFixture(), &fakeRegistryRepo{}, &memRuns{}, &memObjects{}, time.Now)

	if _, err := uc.Execute(context.Background(), "09/2026"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("mês inválido: %v", err)
	}
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
