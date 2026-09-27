package domain

import (
	"testing"
	"time"
)

const portalCommitmentsFixture = `{"cliente":"MUNICÍPIO DE SÃO GONÇALO","empenhos":[
{"nome_razao":"3T Comércio de Materiais e Serviços Ltda. ","documento_formatado":"38.227.436/0001-90","objeto":"Material de Expediente : aquisição  de material.",
 "ds_tp_processo":"Licitação","nr_processo":"2888","ano_processo":"2024","ds_tp_modalidade":"Pregão Eletrônico","nr_modalidade":"PE 90003","ano_modalidade":"2025",
 "tp_documento":"EM","nr_empenho":"1453","ano_empenho":"2025","id_empenho":"124521","dt_empenho":"2025-08-05 00:00:00",
 "vl_empenhado_acu":"R$ 1.270,00","vl_liquidado_acu":"R$ 270,5","vl_pago_acu":"-R$ 3,00"},
{"nome_razao":"Pessoa","documento_formatado":"123.456.789-09","objeto":"x","ds_tp_processo":"Outros/Não aplicável","nr_processo":"27988","ano_processo":"2025",
 "ds_tp_modalidade":"Outros/Não Aplicável","nr_modalidade":"-","ano_modalidade":"-","nr_empenho":"1661","ano_empenho":"2025","id_empenho":"125211",
 "dt_empenho":"2025-10-01 00:00:00","vl_empenhado_acu":"R$ 32.008,20","vl_liquidado_acu":"R$ 32.008,20","vl_pago_acu":"R$ 32.008,20"}],
"totais":{"total_empenhado_acu":"R$ 668.870.191,19","total_liquidado_acu":"R$ 650.608.998,65","total_pago_acu":"R$ 636.550.654,85"}}`

func TestParseMunicipalCommitmentsKeepsCompaniesOnly(t *testing.T) {
	entity := MunicipalEntity{ID: 1, Name: "PREFEITURA MUNICIPAL DE SÃO GONÇALO"}

	got, total, err := ParseMunicipalCommitments([]byte(portalCommitmentsFixture), 2025, entity)

	if err != nil || len(got) != 1 {
		t.Fatalf("empenhos: %+v %v", got, err)
	}
	want := MunicipalCommitment{EntityID: 1, Entity: "PREFEITURA MUNICIPAL DE SÃO GONÇALO", Year: 2025, CommitmentID: 124521, Number: "1453",
		Date: time.Date(2025, 8, 5, 0, 0, 0, 0, time.UTC), CNPJ: "38227436000190", Name: "3T Comércio de Materiais e Serviços Ltda.",
		Object: "Material de Expediente : aquisição de material.", ProcessKind: "Licitação", Process: "2888/2024", Modality: "Pregão Eletrônico PE 90003/2025",
		CommittedCents: 127000, LiquidatedCents: 27050, PaidCents: -300}
	if got[0] != want {
		t.Errorf("empenho:\n%+v\n%+v", got[0], want)
	}
	if total != (MunicipalTotal{Year: 2025, EntityID: 1, Entity: entity.Name, CommittedCents: 66887019119, LiquidatedCents: 65060899865, PaidCents: 63655065485}) {
		t.Errorf("totais: %+v", total)
	}
}

func TestParseMunicipalCommitmentsRejectsBadValues(t *testing.T) {
	for name, body := range map[string]string{
		"json":  `<html>`,
		"valor": `{"empenhos":[{"documento_formatado":"38.227.436/0001-90","id_empenho":"1","dt_empenho":"2025-01-01","vl_empenhado_acu":"R$ x"}],"totais":{}}`,
		"id":    `{"empenhos":[{"documento_formatado":"38.227.436/0001-90","id_empenho":"","dt_empenho":"2025-01-01"}],"totais":{}}`,
	} {
		if _, _, err := ParseMunicipalCommitments([]byte(body), 2025, MunicipalEntity{ID: 1}); err == nil {
			t.Errorf("%s: aceito", name)
		}
	}
}

func TestParseMunicipalEntities(t *testing.T) {
	got, err := ParseMunicipalEntities([]byte(`[{"id_entidade":"1","ds_entidade":"PREFEITURA ","ativo":"1"},{"id_entidade":"2","ds_entidade":"X","ativo":"0"}]`))

	if err != nil || len(got) != 1 || got[0] != (MunicipalEntity{ID: 1, Name: "PREFEITURA"}) {
		t.Errorf("entidades: %+v %v", got, err)
	}
}

func TestWithoutRepeatedCommitmentsKeepsTheFirstRow(t *testing.T) {
	rows := []MunicipalCommitment{{CommitmentID: 1, Object: "despesa"}, {CommitmentID: 2}, {CommitmentID: 1, Object: "inscrição de resto a pagar"}}

	got, repeated := WithoutRepeatedCommitments(rows)

	if repeated != 1 || len(got) != 2 || got[0].Object != "despesa" {
		t.Errorf("veio %+v %d", got, repeated)
	}
}
