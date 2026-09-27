package domain

import (
	"testing"
	"time"
)

func specialRows(t *testing.T) SpecialTransferRows {
	t.Helper()
	var rows SpecialTransferRows
	var err error
	if rows.Plans, err = ParseSpecialPlans([]byte(`[
		{"id_plano_acao":14371,"codigo_plano_acao":"09032022-014371","ano_plano_acao":2022,"situacao_plano_acao":"CIENTE",
		 "nome_parlamentar_emenda_plano_acao":"Soraya Santos","numero_emenda_parlamentar_plano_acao":"202237650006",
		 "codigo_descricao_areas_politicas_publicas_plano_acao":"10-Saúde / 301-Atenção Básica","valor_custeio_plano_acao":0.0,
		 "valor_investimento_plano_acao":2670000.0,"numero_conta_plano_acao":575865909},
		{"id_plano_acao":70146,"codigo_plano_acao":"09032024-070146","ano_plano_acao":2024,"situacao_plano_acao":"IMPEDIDO",
		 "nome_parlamentar_emenda_plano_acao":"Fulano","numero_emenda_parlamentar_plano_acao":"202400010001",
		 "codigo_descricao_areas_politicas_publicas_plano_acao":null,"valor_custeio_plano_acao":500000.5,"valor_investimento_plano_acao":1500000.0}]`)); err != nil {
		t.Fatal(err)
	}
	if rows.Executors, err = ParseSpecialExecutors([]byte(`[{"id_plano_acao":14371,"cnpj_executor":"11884903000107",
		"nome_executor":"FUNDO MUNICIPAL DE SAUDE DE SAO GONCALO","objeto_executor":"Construção da USF Itaúna ",
		"vl_custeio_executor":0.00,"vl_investimento_executor":2670000.00}]`)); err != nil {
		t.Fatal(err)
	}
	if rows.Commitments, err = ParseSpecialCommitments([]byte(`[{"id_empenho":16232,"id_plano_acao":14371,"valor_empenho":2670000.0},
		{"id_empenho":40427,"id_plano_acao":70146,"valor_empenho":2000000.0}]`)); err != nil {
		t.Fatal(err)
	}
	if rows.Documents, err = ParseSpecialDocuments([]byte(`[{"id_dh":36967,"id_empenho":16232,"valor_dh":1620000.0},
		{"id_dh":32051,"id_empenho":16232,"valor_dh":1050000.0},{"id_dh":99,"id_empenho":16232,"valor_dh":7.0}]`)); err != nil {
		t.Fatal(err)
	}
	if rows.Orders, err = ParseSpecialOrders([]byte(`[{"id_dh":32051,"numero_ordem_bancaria":"2022OB802203","data_emissao_ob":"2022-07-01"},
		{"id_dh":36967,"numero_ordem_bancaria":"2023OB804497","data_emissao_ob":"2023-03-29"},
		{"id_dh":99,"numero_ordem_bancaria":null,"data_emissao_ob":null}]`)); err != nil {
		t.Fatal(err)
	}
	if rows.WorkPlans, err = ParseSpecialWorkPlans([]byte(`[{"id_plano_acao":14371,"situacao_plano_trabalho":"CONCLUIDO_NT_TCU",
		"data_fim_execucao_plano_trabalho":"2024-05-01T00:00:00"}]`)); err != nil {
		t.Fatal(err)
	}
	if rows.Reports, err = ParseSpecialReports([]byte(`[
		{"id_plano_acao":14371,"tipo_relatorio_gestao_novo":"Parcial","data_e_hora_relatorio_gestao_novo":"2023-12-30T10:00:00",
		 "valor_executado_relatorio_gestao_novo":100.0,"valor_pendente_relatorio_gestao_novo":2669900.0},
		{"id_plano_acao":14371,"tipo_relatorio_gestao_novo":"Final","data_e_hora_relatorio_gestao_novo":"2024-12-30T16:34:34.305963",
		 "valor_executado_relatorio_gestao_novo":2417780.12,"valor_pendente_relatorio_gestao_novo":252219.88}]`)); err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestBuildSpecialTransfersFollowsThePlanToTheBankOrders(t *testing.T) {
	got := BuildSpecialTransfers(specialRows(t))

	if len(got) != 2 {
		t.Fatalf("planos: %+v", got)
	}
	st := got[0]
	if st.PlanID != 14371 || st.Year != 2022 || st.Author != "Soraya Santos" || st.Area != "10-Saúde / 301-Atenção Básica" || st.ValueCents != 267000000 {
		t.Errorf("plano: %+v", st)
	}
	if len(st.Executors) != 1 || st.Executors[0].Object != "Construção da USF Itaúna" || st.Executors[0].ValueCents != 267000000 {
		t.Errorf("executor: %+v", st.Executors)
	}
	if st.CommittedCents != 267000000 || st.PaidCents != 267000000 || st.LastPaidAt == nil || !st.LastPaidAt.Equal(time.Date(2023, 3, 29, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("pagamento: %+v %v", st, st.LastPaidAt)
	}
	if st.WorkPlanStatus != "CONCLUIDO_NT_TCU" || st.ExecutionEnd == nil || st.ReportKind != "Final" || st.ExecutedCents != 241778012 ||
		st.PendingCents != 25221988 || st.ReportAt == nil || st.ReportAt.Format(time.DateOnly) != "2024-12-30" {
		t.Errorf("execução: %+v", st)
	}
}

func TestBuildSpecialTransfersKeepsABlockedPlanWithoutPayment(t *testing.T) {
	got := BuildSpecialTransfers(specialRows(t))[1]

	if got.Status != "IMPEDIDO" || got.ValueCents != 200000050 || got.CommittedCents != 200000000 || got.PaidCents != 0 ||
		got.LastPaidAt != nil || got.ReportKind != "" || got.Area != "" || len(got.Executors) != 0 {
		t.Errorf("plano impedido: %+v", got)
	}
}

func TestSpecialTransferIDsForTheNextQuery(t *testing.T) {
	rows := specialRows(t)

	if got := SpecialFilterIn(PlanIDs(rows.Plans)); got != "in.(14371,70146)" {
		t.Errorf("planos: %s", got)
	}
	if got := SpecialFilterIn(CommitmentIDs(rows.Commitments)); got != "in.(16232,40427)" {
		t.Errorf("empenhos: %s", got)
	}
	if got := SpecialFilterIn(DocumentIDs(rows.Documents)); got != "in.(36967,32051,99)" {
		t.Errorf("documentos: %s", got)
	}
}

func TestParseSpecialPlansRejectsMalformedJSON(t *testing.T) {
	if _, err := ParseSpecialPlans([]byte(`{"code":"42703"}`)); err == nil {
		t.Error("erro do PostgREST aceito como lista")
	}
}
