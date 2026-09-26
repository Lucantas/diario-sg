package domain

import (
	"testing"
	"time"
)

func TestParseStaffJSONReadsTheTCEAggregates(t *testing.T) {
	body := []byte(`{"SituacoesFuncionais":[{"Ano":2025,"Anomes":"2025/01","UnidadeGestora":"CÂMARA SÃO GONÇALO","Quantidade":183,` +
		`"Remuneracao":1170627.48,"SituacaoFuncional":"Comissionado Extraquadro","Grupo":"Comissionado"},` +
		`{"Ano":2024,"Anomes":"2024/03","UnidadeGestora":"PREFEITURA SÃO GONÇALO","Quantidade":5,"Remuneracao":0.1,` +
		`"SituacaoFuncional":"Efetivo com Função de Confiança","Grupo":"Efetivo (com cargo ou função) "}],"Count":2}`)

	rows, err := ParseStaffJSON(body)

	if err != nil || len(rows) != 2 {
		t.Fatalf("veio %+v %v", rows, err)
	}
	first := rows[0]
	if !first.Month.Equal(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) || first.Unit != "CÂMARA SÃO GONÇALO" || first.Headcount != 183 ||
		first.RemunerationCents != 117062748 || first.Group != "Comissionado" || first.Situation != "Comissionado Extraquadro" {
		t.Fatalf("primeira linha: %+v", first)
	}
	if rows[1].Group != "Efetivo (com cargo ou função)" || rows[1].RemunerationCents != 10 {
		t.Fatalf("segunda linha: %+v", rows[1])
	}
}

func TestParseStaffJSONRejectsABadMonth(t *testing.T) {
	if _, err := ParseStaffJSON([]byte(`{"SituacoesFuncionais":[{"Anomes":"janeiro"}]}`)); err == nil {
		t.Fatal("esperava erro")
	}
}
