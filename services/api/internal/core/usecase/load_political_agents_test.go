package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeAgentSource struct {
	months     []string
	years      []int
	councilYrs []int
	failMonth  string
	failYear   int
}

func (f *fakeAgentSource) PrefeituraPay(_ context.Context, year, month int) ([]byte, error) {
	key := fmt.Sprintf("%d-%02d", year, month)
	f.months = append(f.months, key)
	if key == f.failMonth {
		return nil, errors.New("status 500")
	}
	return []byte(fmt.Sprintf(`{"success":true,"data":[
		{"nome":"PREFEITO X","funcao":"PREFEITO","organograma":"01 - GABINETE DO PREFEITO","remuneracao":23813.22},
		{"nome":"SERVIDOR Y","funcao":"PROFESSOR","organograma":"30 - EDUCACAO","remuneracao":%d}]}`, 1000+month)), nil
}

func (f *fakeAgentSource) CamaraPay(_ context.Context, year int) ([]byte, error) {
	f.years = append(f.years, year)
	if year == f.failYear {
		return []byte(`{"mensagem":"Erro interno"}`), nil
	}
	var rows []string
	for m := 1; m <= 12; m++ {
		rows = append(rows, fmt.Sprintf(`{"ano":"%d","mes":"%02d","nome":"VEREADOR Z","cargo":"VEREADOR","regime":"Agente Político","secretaria":"VEREADOR ZE",
			"nome_rem01":"Vencimentos","valor_rem01":21840.43,"nome_rem02":"Descontos","valor_rem02":5000,"nome_rem03":"Líquido","valor_rem03":16840.43}`, year, m))
	}
	return []byte("[" + strings.Join(rows, ",") + "]"), nil
}

func (f *fakeAgentSource) Councillors(_ context.Context, year int) ([]byte, error) {
	f.councilYrs = append(f.councilYrs, year)
	return []byte(`{"Parametros":[{"Nome_Vereador":"ZE","Nome":"VEREADOR Z","Partido":"PX","Legislatura":4,"Situacao":"Ativo"}]}`), nil
}

type fakeAgentRepo struct {
	saved map[string]domain.PoliticalAgentLoad
}

func (r *fakeAgentRepo) Ready(context.Context) error { return nil }
func (r *fakeAgentRepo) SavePoliticalAgents(_ context.Context, load domain.PoliticalAgentLoad) error {
	if r.saved == nil {
		r.saved = map[string]domain.PoliticalAgentLoad{}
	}
	r.saved[load.Body] = load
	return nil
}

func agentDay(y int, m time.Month) time.Time { return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC) }

func TestLoadPoliticalAgentsDefaultsToTheLastThreeMonths(t *testing.T) {
	src, repo, raw := &fakeAgentSource{}, &fakeAgentRepo{}, &memObjects{}
	now := func() time.Time { return time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC) }

	run, err := NewLoadPoliticalAgents(src, repo, &memRuns{}, raw, now).Execute(context.Background(), time.Time{}, time.Time{})

	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(src.months, ",") != "2026-07,2026-08,2026-09" || len(src.years) != 1 || src.years[0] != 2026 || len(src.councilYrs) != 1 {
		t.Errorf("pedidos: %v %v %v", src.months, src.years, src.councilYrs)
	}
	prefeitura, camara := repo.saved[domain.BodyPrefeitura], repo.saved[domain.BodyCamara]
	if !prefeitura.From.Equal(agentDay(2026, 7)) || !prefeitura.To.Equal(agentDay(2026, 9)) || len(prefeitura.Pay) != 3 || len(camara.Pay) != 3 || len(camara.Councillors) != 1 {
		t.Errorf("carga: %+v", repo.saved)
	}
	for _, p := range append(prefeitura.Pay, camara.Pay...) {
		if p.Name == "SERVIDOR Y" {
			t.Errorf("servidor comum gravado: %+v", p)
		}
	}
	if run.Found != 6 || run.Stored != 6 || run.Source != domain.SourcePoliticalAgents {
		t.Errorf("coleta: %+v", run)
	}
	archived := gunzip(t, raw.data["raw/agentes_politicos/2026/09/26/prefeitura.json.gz"])
	if strings.Contains(archived, "SERVIDOR Y") || !strings.Contains(archived, "PREFEITO X") {
		t.Errorf("arquivo bruto: %s", archived)
	}
	var manifest agentArchive
	if err := json.Unmarshal(raw.data["raw/agentes_politicos/2026/09/26/prefeitura.manifest.json"], &manifest); err != nil || len(manifest.Sources) != 3 || manifest.Rows != 3 {
		t.Errorf("manifesto: %+v %v", manifest, err)
	}
}

func TestLoadPoliticalAgentsStartsAtTheFirstPublishedMonths(t *testing.T) {
	src, repo := &fakeAgentSource{}, &fakeAgentRepo{}
	now := func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }

	if _, err := NewLoadPoliticalAgents(src, repo, &memRuns{}, &memObjects{}, now).Execute(context.Background(), agentDay(2009, 1), agentDay(2017, 2)); err != nil {
		t.Fatal(err)
	}

	if src.months[0] != "2010-10" || src.years[0] != 2017 || len(src.years) != 1 || !repo.saved[domain.BodyPrefeitura].From.Equal(agentDay(2010, 10)) {
		t.Errorf("início: %v %v %v", src.months[:1], src.years, repo.saved[domain.BodyPrefeitura].From)
	}
	if n := len(repo.saved[domain.BodyCamara].Pay); n != 2 {
		t.Errorf("meses da Câmara fora do período: %d", n)
	}
}

func TestLoadPoliticalAgentsPrefeituraFailureStillSavesTheCamara(t *testing.T) {
	src, repo, runs := &fakeAgentSource{failMonth: "2026-08"}, &fakeAgentRepo{}, &memRuns{}
	now := func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }

	_, err := NewLoadPoliticalAgents(src, repo, runs, &memObjects{}, now).Execute(context.Background(), time.Time{}, time.Time{})

	_, prefeituraSaved := repo.saved[domain.BodyPrefeitura]
	if err == nil || prefeituraSaved || len(repo.saved[domain.BodyCamara].Pay) != 3 || len(runs.runs) != 1 || !strings.Contains(runs.runs[0].Error, domain.BodyPrefeitura) {
		t.Errorf("falha: %v %+v %+v", err, repo.saved, runs.runs)
	}
}

func TestLoadPoliticalAgentsCamaraFailureStillSavesThePrefeitura(t *testing.T) {
	src, repo := &fakeAgentSource{failYear: 2026}, &fakeAgentRepo{}
	now := func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }

	run, err := NewLoadPoliticalAgents(src, repo, &memRuns{}, &memObjects{}, now).Execute(context.Background(), time.Time{}, time.Time{})

	_, camaraSaved := repo.saved[domain.BodyCamara]
	if err == nil || camaraSaved || len(repo.saved[domain.BodyPrefeitura].Pay) != 3 || run.Stored != 3 {
		t.Errorf("falha: %v %+v %+v", err, repo.saved, run)
	}
}

func TestLoadPoliticalAgentsRejectsFutureMonths(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }

	_, err := NewLoadPoliticalAgents(&fakeAgentSource{}, &fakeAgentRepo{}, &memRuns{}, &memObjects{}, now).Execute(context.Background(), agentDay(2026, 9), agentDay(2026, 10))

	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("mês futuro: %v", err)
	}
}

type fakeAgentReader struct {
	pay         []domain.AgentPay
	councillors []domain.Councillor
}

func (f fakeAgentReader) AgentPay(context.Context) ([]domain.AgentPay, error) { return f.pay, nil }
func (f fakeAgentReader) Councillors(context.Context) ([]domain.Councillor, error) {
	return f.councillors, nil
}

func TestGetPoliticalAgentsFiltersByRoleAndName(t *testing.T) {
	reader := fakeAgentReader{pay: []domain.AgentPay{
		{Body: domain.BodyPrefeitura, Month: agentDay(2026, 8), Name: "JOÃO PREFEITO", NameKey: "JOAO PREFEITO", Role: domain.RolePrefeito, GrossCents: 1},
		{Body: domain.BodyCamara, Month: agentDay(2026, 8), Name: "ANA VEREADORA", NameKey: "ANA VEREADORA", Role: domain.RoleVereador, Office: "VEREADORA ANINHA", GrossCents: 1},
		{Body: domain.BodyCamara, Month: agentDay(2017, 1), Name: "ANA VEREADORA", NameKey: "ANA VEREADORA", Role: domain.RoleVereador, Office: "VEREADORA ANINHA", GrossCents: 1},
	}}
	uc := NewGetPoliticalAgents(reader)

	report, err := uc.Execute(context.Background(), domain.RoleVereador, "aninha")

	if err != nil || len(report.Agents) != 1 || report.Agents[0].Name != "ANA VEREADORA" || len(report.Norms) == 0 {
		t.Fatalf("filtro: %+v %v", report, err)
	}
	if len(report.Coverage) != 2 || !report.Coverage[1].From.Equal(agentDay(2017, 1)) {
		t.Errorf("cobertura: %+v", report.Coverage)
	}
	if _, err := uc.Execute(context.Background(), "rei", ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("papel inválido: %v", err)
	}
}
