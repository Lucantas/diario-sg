package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestSortLatestDaysPutsPrefeituraFirstAndBusiestTypesFirst(t *testing.T) {
	day := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	days := []LatestDay{
		{Source: SourceDiarioCamara, Day: day, Types: []TypeTotal{{Type: ActLei, Acts: 1}}},
		{Source: SourceDiarioPrefeitura, Day: day, Types: []TypeTotal{
			{Type: ActContrato, Acts: 3, ValueCents: 210000000},
			{Type: ActNomeacao, Acts: 14},
			{Type: ActDecreto, Acts: 3},
		}},
	}

	SortLatestDays(days)

	if days[0].Source != SourceDiarioPrefeitura || days[1].Source != SourceDiarioCamara {
		t.Fatalf("Prefeitura vem antes da Câmara: %+v", days)
	}
	want := []ActType{ActNomeacao, ActContrato, ActDecreto}
	var got []ActType
	for _, tt := range days[0].Types {
		got = append(got, tt.Type)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tipos em ordem de quantidade, empate pelo nome do tipo: %v", got)
	}
}
