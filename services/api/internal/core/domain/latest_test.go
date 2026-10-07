package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestSortLatestEditionsPutsPrefeituraAndRegularEditionsFirst(t *testing.T) {
	day := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	editions := []LatestEdition{
		{Source: SourceDiarioCamara, Day: day, EditionNumber: "138"},
		{Source: SourceDiarioPrefeitura, Day: day, EditionNumber: "1413", IsExtra: true},
		{Source: SourceDiarioPrefeitura, Day: day, EditionNumber: "1412", Types: []TypeTotal{
			{Type: ActContrato, Acts: 3, ValueCents: 210000000},
			{Type: ActNomeacao, Acts: 14},
			{Type: ActDecreto, Acts: 3},
		}},
	}

	SortLatestEditions(editions)

	var order []string
	for _, e := range editions {
		order = append(order, e.EditionNumber)
	}
	if !reflect.DeepEqual(order, []string{"1412", "1413", "138"}) {
		t.Fatalf("Prefeitura antes da Câmara, edição normal antes da extra: %v", order)
	}
	var types []ActType
	for _, tt := range editions[0].Types {
		types = append(types, tt.Type)
	}
	if !reflect.DeepEqual(types, []ActType{ActNomeacao, ActContrato, ActDecreto}) {
		t.Errorf("tipos em ordem de quantidade, empate pelo nome do tipo: %v", types)
	}
}

func TestValidGazetteID(t *testing.T) {
	if !ValidGazetteID("3f2b8c1e-4d5a-4b6c-8d7e-9f0a1b2c3d4e") || ValidGazetteID("1412") || ValidGazetteID("'; DROP") {
		t.Error("só UUID é id de edição")
	}
}
