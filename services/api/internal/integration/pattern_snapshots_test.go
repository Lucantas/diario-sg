//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

func TestPatternSnapshotRoundTripsTheComputedPatterns(t *testing.T) {
	_, db := newServerFor(t, semedHomologacao2025)
	ctx := context.Background()
	indexAt(t, db, time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC),
		"PORTARIA Nº 7/2025\nO PREFEITO resolve NOMEAR Mariana Barbosa de Souza para Assessora da Secretaria de Obras.")
	loadRegistry(t, postgres.NewRegistryRepo(db), postgres.NewFetchRunRepo(db),
		registryShare{month: "2026-09", name: "F.P. VIEIRA ENGENHARIA LTDA", partner: "MARIANA BARBOSA DE SOUZA"})
	if err := postgres.RefreshPartnerAppointments(ctx, db); err != nil {
		t.Fatal(err)
	}
	reports, acts, err := usecase.NewListPatterns(postgres.NewPatternRepo(db), postgres.NewSupplierPatternRepo(db)).Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repo := postgres.NewPatternSnapshotRepo(db)

	if err := repo.SavePatternSnapshot(ctx, domain.PatternSnapshot{Fingerprint: "v1|dados", Reports: reports, Acts: acts}); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := repo.LoadPatternSnapshot(ctx, "v1|dados")

	if err != nil || !ok || len(loaded.Acts) == 0 {
		t.Fatalf("esperava ler o que foi guardado: ok=%v atos=%d %v", ok, len(loaded.Acts), err)
	}
	if got, want := asJSON(t, loaded.Reports, loaded.Acts), asJSON(t, reports, acts); got != want {
		t.Errorf("o guardado volta igual ao calculado:\nveio  %s\nquero %s", got, want)
	}
	if _, ok, err := repo.LoadPatternSnapshot(ctx, "v1|outros-dados"); ok || err != nil {
		t.Errorf("outra impressão digital não lê o guardado: ok=%v %v", ok, err)
	}
}

func TestPatternInputsFingerprintChangesOnlyWhenTheInputsChange(t *testing.T) {
	_, db := newServerFor(t, semedHomologacao2025)
	ctx := context.Background()
	repo := postgres.NewPatternSnapshotRepo(db)
	before := fingerprint(t, repo)

	if month := time.Now().UTC().Format("2006-01"); !strings.HasPrefix(before, "mes:"+month+",") {
		t.Fatalf("o pico de nomeações ignora o mês corrente, então a impressão digital leva o mês: %s", before)
	}
	if again := fingerprint(t, repo); again != before {
		t.Fatalf("sem escrita a impressão digital não muda: %s -> %s", before, again)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO political_agent_pay (body, month, name, name_key, role, office, gross_cents)
		VALUES ('prefeitura', '2025-01-01', 'Fulana de Tal', 'FULANA DE TAL', 'secretario', 'SEMOBI', 100)`); err != nil {
		t.Fatal(err)
	}
	inserted := fingerprint(t, repo)
	if inserted == before {
		t.Fatal("um registro novo muda a impressão digital")
	}
	if _, err := db.ExecContext(ctx, `UPDATE political_agent_pay SET gross_cents = 200`); err != nil {
		t.Fatal(err)
	}
	if updated := fingerprint(t, repo); updated == inserted {
		t.Error("uma alteração sem registro novo também muda a impressão digital")
	}
	last := fingerprint(t, repo)
	if err := repo.SavePatternSnapshot(ctx, domain.PatternSnapshot{Fingerprint: "v1|" + last}); err != nil {
		t.Fatal(err)
	}
	if fingerprint(t, repo) != last {
		t.Error("guardar os padrões não muda a impressão digital dos dados")
	}
}

func fingerprint(t *testing.T, repo *postgres.PatternSnapshotRepo) string {
	t.Helper()
	fp, err := repo.PatternInputsFingerprint(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return fp
}

func asJSON(t *testing.T, reports []domain.PatternReport, acts map[string]domain.ActHit) string {
	t.Helper()
	b, err := json.Marshal(struct {
		Reports []domain.PatternReport
		Acts    map[string]domain.ActHit
	}{reports, acts})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
