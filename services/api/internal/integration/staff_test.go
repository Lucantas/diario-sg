//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

type staffAPI map[int]string

func (a staffAPI) Staff(_ context.Context, year int) ([]byte, error) {
	return []byte(a[year]), nil
}

func staffJSON(rows ...string) string {
	out := `{"SituacoesFuncionais":[`
	for i, r := range rows {
		if i > 0 {
			out += ","
		}
		out += r
	}
	return out + `]}`
}

func staffRecord(month, unit, situation, group string, headcount int, remuneration float64) string {
	return fmt.Sprintf(`{"Anomes":%q,"UnidadeGestora":%q,"Quantidade":%d,"Remuneracao":%.2f,"SituacaoFuncional":%q,"Grupo":%q}`,
		month, unit, headcount, remuneration, situation, group)
}

func loadStaff(t *testing.T, db *sql.DB, api staffAPI, from, to int) {
	t.Helper()
	now := func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }
	uc := usecase.NewLoadStaff(api, postgres.NewStaffRepo(db), postgres.NewFetchRunRepo(db), discardObjects{}, now)
	if _, err := uc.Execute(context.Background(), from, to); err != nil {
		t.Fatal(err)
	}
}

func TestStaffReplacesTheYearsRead(t *testing.T) {
	_, db := newServerFor(t, semedHomologacao2025)
	repo := postgres.NewStaffRepo(db)

	loadStaff(t, db, staffAPI{
		2025: staffJSON(staffRecord("2025/01", "PREFEITURA SÃO GONÇALO", "Efetivo - Estatutário", "Efetivo", 100, 500000)),
		2026: staffJSON(staffRecord("2026/01", "PREFEITURA SÃO GONÇALO", "Efetivo - Estatutário", "Efetivo", 90, 450000),
			staffRecord("2026/01", "CÂMARA SÃO GONÇALO", "Comissionado Extraquadro", "Comissionado", 180, 1170627.48)),
	}, 2025, 2026)
	loadStaff(t, db, staffAPI{2026: staffJSON(staffRecord("2026/02", "CÂMARA SÃO GONÇALO", "Comissionado Extraquadro", "Comissionado", 181, 1000))}, 2026, 2026)

	all, err := repo.StaffRows(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].Month.Year() != 2025 || all[1].Month.Month() != time.February || all[1].RemunerationCents != 100000 {
		t.Fatalf("linhas: %+v", all)
	}
	units, err := repo.StaffUnits(context.Background())
	if err != nil || len(units) != 2 || units[0] != "CÂMARA SÃO GONÇALO" {
		t.Fatalf("unidades: %v %v", units, err)
	}
	camara, err := repo.StaffRows(context.Background(), "CÂMARA SÃO GONÇALO")
	if err != nil || len(camara) != 1 || camara[0].Headcount != 181 {
		t.Fatalf("Câmara: %+v %v", camara, err)
	}
}

func TestStaffPanelRouteGroupsTheMonths(t *testing.T) {
	srv, db := newServerFor(t, semedHomologacao2025)
	loadStaff(t, db, staffAPI{2025: staffJSON(
		staffRecord("2025/01", "PREFEITURA SÃO GONÇALO", "Efetivo - Estatutário", "Efetivo", 100, 5000),
		staffRecord("2025/01", "CÂMARA SÃO GONÇALO", "Comissionado Extraquadro", "Comissionado", 180, 1000.5),
		staffRecord("2025/02", "CÂMARA SÃO GONÇALO", "Comissionado Extraquadro", "Comissionado", 181, 1000))}, 2025, 2025)

	var panel struct {
		Units  []string `json:"units"`
		Source string   `json:"diario_source"`
		Groups []struct {
			Label string `json:"label"`
		} `json:"groups"`
		Months []struct {
			Month             string `json:"month"`
			Headcount         int    `json:"headcount"`
			RemunerationCents int64  `json:"remuneration_cents"`
			Groups            []struct {
				Headcount int `json:"headcount"`
			} `json:"groups"`
		} `json:"months"`
	}
	getJSON(t, srv.URL+"/v1/panels/staff", &panel)
	if len(panel.Units) != 2 || panel.Source != "diario_prefeitura" || len(panel.Groups) != 2 || panel.Groups[0].Label != "Efetivos" ||
		len(panel.Months) != 2 || panel.Months[1].Month != "2025-01" || panel.Months[1].Headcount != 280 ||
		panel.Months[1].RemunerationCents != 600050 || panel.Months[1].Groups[1].Headcount != 180 {
		t.Fatalf("painel: %+v", panel)
	}

	getJSON(t, srv.URL+"/v1/panels/staff?unit=C%C3%82MARA+S%C3%83O+GON%C3%87ALO", &panel)
	if panel.Source != "diario_camara" || len(panel.Groups) != 1 || panel.Months[0].Headcount != 181 {
		t.Fatalf("Câmara: %+v", panel)
	}

	resp, err := http.Get(srv.URL + "/v1/panels/staff?unit=OUTRA")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unidade desconhecida: %d", resp.StatusCode)
	}
}
