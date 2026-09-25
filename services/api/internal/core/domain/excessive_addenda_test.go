package domain

import (
	"testing"
	"time"
)

const (
	fmsAddendumBody   = "OBJETO: O presente Termo Aditivo tem por objetivo o acréscimo equivalente a 46,37% do valor inicial contratado, o que perfaz um total de R$ 240.802,53"
	semedAddendumBody = "Objeto: Fica rerratificado o contrato PMSG nº 019/2016, referente contratação de empresa de engenharia para reforma e construção de novas instalações na escola municipal Professora Marlucy Salles Almeida, com o acréscimo no valor de R$ 479.193,00, que equivale a 49,13% do valor inicial do contrato"
)

func addendumAct(id, key, organ string, day time.Time, title, body string, bp int) AddendumAct {
	return AddendumAct{ActID: id, ContractKeys: []string{key}, Organ: organ, PublishedAt: day, Title: title, Body: body, IncreaseBP: bp}
}

func TestFindExcessiveAddendaFlagsIncreaseAboveTheLimit(t *testing.T) {
	acts := []AddendumAct{
		addendumAct("fms", "7/2015", "FMS", civilDate(2015, 9, 8), "EXTRATO DE TERMO ADITIVO DE CONTRATO", fmsAddendumBody, 4637),
		addendumAct("semed", "19/2016", "SEMED", civilDate(2016, 9, 9), "EXTRATO DE TERMO ADITIVO", "Primeiro Termo aditivo ao Contrato n° 019/2016 "+semedAddendumBody, 4913),
	}

	got := FindExcessiveAddenda(acts)

	if len(got) != 1 || got[0].ContractKey != "7/2015" || got[0].Organ != "FMS" || got[0].TotalBP != 4637 || got[0].LimitBP != 2500 {
		t.Fatalf("só o contrato da FMS deveria acionar (a reforma da SEMED tem limite de 50%%): %+v", got)
	}
}

func TestFindExcessiveAddendaSumsDistinctAddendaOfTheSameContract(t *testing.T) {
	acts := []AddendumAct{
		addendumAct("1", "10/2020", "FMSSG", civilDate(2020, 3, 1), "EXTRATO DO PRIMEIRO TERMO ADITIVO", "acréscimo de 15%", 1500),
		addendumAct("1-rep", "10/2020", "FMS", civilDate(2020, 3, 9), "EXTRATO DO PRIMEIRO TERMO ADITIVO", "acréscimo de 15%", 1500),
		addendumAct("2", "10/2020", "FMS", civilDate(2020, 9, 1), "EXTRATO DO SEGUNDO TERMO ADITIVO", "acréscimo de 15%", 1500),
		addendumAct("outro", "10/2020", "SEMED", civilDate(2020, 9, 1), "EXTRATO DO SEGUNDO TERMO ADITIVO", "acréscimo de 20%", 2000),
	}

	got := FindExcessiveAddenda(acts)

	if len(got) != 1 || got[0].TotalBP != 3000 || len(got[0].Acts) != 2 || got[0].Acts[0].ActID != "1" || got[0].Acts[1].ActID != "2" {
		t.Fatalf("esperava dois aditivos distintos somando 30%%: %+v", got)
	}
}

func TestFindExcessiveAddendaCountsTheSamePercentOnceWithoutOrdinal(t *testing.T) {
	acts := []AddendumAct{
		addendumAct("a", "19/2015", "FMS", civilDate(2015, 4, 30), "EXTRATO DE TERMO ADITIVO", "acréscimo de 15,66%", 1566),
		addendumAct("b", "19/2015", "FMS", civilDate(2015, 5, 4), "EXTRATO DE TERMO ADITIVO", "acréscimo de 15,66%", 1566),
	}

	if got := FindExcessiveAddenda(acts); len(got) != 0 {
		t.Fatalf("a mesma publicação repetida não soma: %+v", got)
	}
}
