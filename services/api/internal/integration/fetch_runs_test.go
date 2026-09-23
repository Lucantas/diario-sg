//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

func TestFetchRunsAreIdempotentAndShownInSources(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)
	ctx := context.Background()
	start := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
	uc := usecase.NewRecordFetchRun(postgres.NewFetchRunRepo(db))
	older := domain.FetchRun{ID: "11111111-1111-4111-8111-111111111111", Source: domain.SourceDiarioPrefeitura,
		RequestedFrom: start.AddDate(0, 0, -6), RequestedTo: start.AddDate(0, 0, -3), Found: 1, Stored: 1,
		StartedAt: start.Add(-72 * time.Hour), FinishedAt: start.Add(-72*time.Hour + time.Minute)}
	latest := domain.FetchRun{ID: "22222222-2222-4222-8222-222222222222", Source: domain.SourceDiarioPrefeitura,
		RequestedFrom: start.AddDate(0, 0, -3), RequestedTo: start, Found: 3, Stored: 2, Failed: 1,
		Error: "edição 1780: timeout", StartedAt: start, FinishedAt: start.Add(2 * time.Minute)}
	camara := domain.FetchRun{ID: "33333333-3333-4333-8333-333333333333", Source: domain.SourceDiarioCamara,
		RequestedFrom: start, RequestedTo: start, Found: 1, Stored: 1, StartedAt: start.Add(time.Hour), FinishedAt: start.Add(time.Hour + time.Minute)}
	for _, run := range []domain.FetchRun{latest, older, latest, camara} {
		if err := uc.Execute(ctx, run); err != nil {
			t.Fatal(err)
		}
	}
	var stored int
	if err := db.QueryRow(`SELECT count(*) FROM fetch_runs`).Scan(&stored); err != nil || stored != 3 {
		t.Fatalf("a mesma coleta entregue duas vezes deveria virar uma linha: %d %v", stored, err)
	}

	code, key := issueKey(t, srv.URL, "")
	if code != 201 {
		t.Fatalf("emissão da chave: %d", code)
	}
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	sources, _ := call[struct {
		Sources []struct {
			LastRun *struct {
				FinishedAt string `json:"em"`
				Found      int    `json:"encontradas"`
				Stored     int    `json:"gravadas"`
				Failed     int    `json:"falhas"`
				Error      string `json:"erro"`
			} `json:"ultima_coleta_tentada"`
		} `json:"fontes"`
	}](t, session, "fontes", nil)
	if len(sources.Sources) != 2 || sources.Sources[0].LastRun == nil || sources.Sources[1].LastRun == nil || sources.Sources[1].LastRun.Found != 1 {
		t.Fatalf("fontes sem a última coleta: %+v", sources)
	}
	got := *sources.Sources[0].LastRun
	if got.FinishedAt != "2026-09-23T12:02:00-03:00" || got.Found != 3 || got.Stored != 2 || got.Failed != 1 || got.Error != latest.Error {
		t.Fatalf("última coleta errada: %+v", got)
	}
}
