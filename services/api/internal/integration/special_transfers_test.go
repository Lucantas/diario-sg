//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestSpecialTransfersRouteShowsAuthorObjectAndExecution(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)
	paid := time.Date(2023, 3, 29, 0, 0, 0, 0, time.UTC)
	repo := postgres.NewSpecialTransferRepo(db)
	old := []domain.SpecialTransfer{{PlanID: 1, Code: "velho", Year: 2020, Executors: []domain.SpecialExecutor{{Object: "velho"}}}}
	if err := repo.ReplaceSpecialTransfers(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	fresh := []domain.SpecialTransfer{
		{PlanID: 14371, Code: "09032022-014371", Year: 2022, Status: "CIENTE", Author: "Soraya Santos", ValueCents: 267000000, CommittedCents: 267000000,
			PaidCents: 267000000, LastPaidAt: &paid, ReportKind: "Final", ReportAt: &paid, ExecutedCents: 241778012, PendingCents: 25221988,
			Executors: []domain.SpecialExecutor{{CNPJ: "11884903000107", Name: "FUNDO MUNICIPAL DE SAUDE", Object: "USF Itaúna", ValueCents: 267000000}}},
		{PlanID: 70146, Code: "09032024-070146", Year: 2024, Status: "IMPEDIDO", Author: "Fulano", ValueCents: 200000000},
	}
	if err := repo.ReplaceSpecialTransfers(context.Background(), fresh); err != nil {
		t.Fatal(err)
	}

	var out struct {
		Special []struct {
			PlanID     int64   `json:"plan_id"`
			Author     string  `json:"author"`
			PaidCents  int64   `json:"paid_cents"`
			LastPaidAt *string `json:"last_paid_at"`
			ReportKind string  `json:"report_kind"`
			Executed   int64   `json:"executed_cents"`
			Executors  []struct {
				Object string `json:"object"`
			} `json:"executors"`
		} `json:"special_transfers"`
	}
	getJSON(t, srv.URL+"/v1/federal", &out)

	if len(out.Special) != 2 || out.Special[0].PlanID != 70146 || out.Special[0].LastPaidAt != nil || len(out.Special[0].Executors) != 0 {
		t.Fatalf("transferências: %+v", out.Special)
	}
	st := out.Special[1]
	if st.Author != "Soraya Santos" || st.PaidCents != 267000000 || st.LastPaidAt == nil || *st.LastPaidAt != "2023-03-29" || st.ReportKind != "Final" ||
		st.Executed != 241778012 || len(st.Executors) != 1 || st.Executors[0].Object != "USF Itaúna" {
		t.Errorf("plano pago: %+v", st)
	}
}
