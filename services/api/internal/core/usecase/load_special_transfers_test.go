package usecase

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type fakeTransferegov struct {
	tables map[string]string
	asked  []string
}

func (f *fakeTransferegov) Rows(_ context.Context, table string, filter url.Values) ([]byte, error) {
	f.asked = append(f.asked, table+"?"+filter.Encode())
	body, ok := f.tables[table]
	if !ok {
		return nil, errors.New("fora do ar")
	}
	return []byte(body), nil
}

type fakeSpecialRepo struct{ saved []domain.SpecialTransfer }

func (f *fakeSpecialRepo) Ready(context.Context) error { return nil }

func (f *fakeSpecialRepo) ReplaceSpecialTransfers(_ context.Context, t []domain.SpecialTransfer) error {
	f.saved = t
	return nil
}

func transferegovFixture() map[string]string {
	return map[string]string{
		tablePlans:       `[{"id_plano_acao":14371,"ano_plano_acao":2022,"situacao_plano_acao":"CIENTE","valor_investimento_plano_acao":2670000.0}]`,
		tableExecutors:   `[{"id_plano_acao":14371,"objeto_executor":"USF Itaúna","vl_investimento_executor":2670000.0}]`,
		tableWorkPlans:   `[]`,
		tableReports:     `[]`,
		tableCommitments: `[{"id_empenho":16232,"id_plano_acao":14371,"valor_empenho":2670000.0}]`,
		tableDocuments:   `[{"id_dh":32051,"id_empenho":16232,"valor_dh":1050000.0}]`,
		tableOrders:      `[{"id_dh":32051,"numero_ordem_bancaria":"2022OB802203","data_emissao_ob":"2022-07-01"}]`,
	}
}

func TestLoadSpecialTransfersFollowsThePlansToThePayments(t *testing.T) {
	src := &fakeTransferegov{tables: transferegovFixture()}
	repo, runs, raw := &fakeSpecialRepo{}, &memRuns{}, &memObjects{}

	run, err := NewLoadSpecialTransfers(src, repo, runs, raw, staffNow).Execute(context.Background())

	if err != nil || run.Stored != 1 || len(repo.saved) != 1 || repo.saved[0].PaidCents != 105000000 || repo.saved[0].Executors[0].Object != "USF Itaúna" {
		t.Fatalf("veio %+v %v %+v", run, err, repo.saved)
	}
	want := []string{
		tablePlans + "?cnpj_beneficiario_plano_acao=eq.28636579000100",
		tableExecutors + "?id_plano_acao=in.%2814371%29",
		tableWorkPlans + "?id_plano_acao=in.%2814371%29",
		tableReports + "?id_plano_acao=in.%2814371%29",
		tableCommitments + "?id_plano_acao=in.%2814371%29",
		tableDocuments + "?id_empenho=in.%2816232%29",
		tableOrders + "?id_dh=in.%2832051%29",
	}
	if len(src.asked) != len(want) {
		t.Fatalf("pedidos: %v", src.asked)
	}
	for i := range want {
		if src.asked[i] != want[i] {
			t.Errorf("pedido %d: %s", i, src.asked[i])
		}
	}
	if _, ok := raw.data["raw/transferegov_especiais/2026/09/26/"+tableOrders+".json.gz"]; !ok {
		t.Errorf("bruto: %v", raw.names)
	}
}

func TestLoadSpecialTransfersStopsWhenThereIsNoPlan(t *testing.T) {
	src := &fakeTransferegov{tables: map[string]string{tablePlans: `[]`}}
	repo := &fakeSpecialRepo{}

	run, err := NewLoadSpecialTransfers(src, repo, &memRuns{}, &memObjects{}, staffNow).Execute(context.Background())

	if err != nil || run.Stored != 0 || len(src.asked) != 1 {
		t.Fatalf("veio %+v %v %v", run, err, src.asked)
	}
}

func TestLoadSpecialTransfersSavesNothingWhenATableFails(t *testing.T) {
	tables := transferegovFixture()
	delete(tables, tableOrders)
	repo, runs := &fakeSpecialRepo{}, &memRuns{}

	_, err := NewLoadSpecialTransfers(&fakeTransferegov{tables: tables}, repo, runs, &memObjects{}, staffNow).Execute(context.Background())

	if err == nil || repo.saved != nil || runs.runs[0].Error == "" {
		t.Fatalf("veio %v %+v", err, repo.saved)
	}
}
