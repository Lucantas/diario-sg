package domain

import (
	"testing"
	"time"
)

func TestSplitDispensaFinding(t *testing.T) {
	s := SplitDispensa{Category: DispensaGoods, CNPJ: "53775862000152", Year: 2020, TotalCents: 2891160, LimitCents: 1760000, Contracts: []DispensaContract{
		{Processes: []DispensaProcess{{"215772019", "21.577/2019"}}, FirstPublished: civilDate(2020, 2, 19), ValueCents: 1186200, Organs: []string{"SEMMA"}, ActIDs: []string{"a"}},
		{Processes: []DispensaProcess{{"536392018", "53.639/2018"}, {"566392018", "56.639/2018"}}, FirstPublished: civilDate(2020, 5, 22), ValueCents: 1704960, Organs: []string{"SEMAD"}, ActIDs: []string{"b", "c"}},
	}}

	f := SplitDispensaFinding(s)

	if f.Title != "CNPJ 53.775.862/0001-52 em 2020: 2 dispensas de compras e serviços somam R$ 28.911,60, acima do limite de R$ 17.600,00" {
		t.Errorf("título: %q", f.Title)
	}
	if f.Detail != "Processo 21.577/2019 (SEMMA), 19/02/2020: R$ 11.862,00. Processos 53.639/2018 e 56.639/2018 (SEMAD), 22/05/2020: R$ 17.049,60." {
		t.Errorf("detalhe: %q", f.Detail)
	}
	if len(f.ActIDs) != 3 || f.Search != nil {
		t.Errorf("atos: %+v", f)
	}
}

func TestElectionPeakFinding(t *testing.T) {
	p := HiringPeak{Type: ActNomeacao, Year: 2020, Month: time.August, Count: 166, BaselineMedian: 107, BaselineYears: 13, Election: civilDate(2020, 11, 15)}

	f := ElectionPeakFinding(p)

	if f.Title != "Agosto de 2020: 166 atos de nomeação" ||
		f.Detail != "Mediana de agosto nos 13 anos sem eleição municipal: 107. Eleição em 15/11/2020." {
		t.Errorf("caso inesperado: %+v", f)
	}
	if f.Search == nil || f.Search.Type != ActNomeacao || f.Search.Source != SourceDiarioPrefeitura ||
		!f.Search.From.Equal(civilDate(2020, 8, 1)) || !f.Search.To.Equal(civilDate(2020, 8, 31)) {
		t.Errorf("busca inesperada: %+v", f.Search)
	}
}

func TestExcessiveAddendumFinding(t *testing.T) {
	e := ExcessiveAddendum{ContractKey: "7/2015", Organ: "FMS", TotalBP: 4637, LimitBP: 2500, Acts: []AddendumAct{
		{ActID: "a", Title: "EXTRATO DO PRIMEIRO TERMO ADITIVO", PublishedAt: civilDate(2015, 9, 8), IncreaseBP: 4637},
	}}

	f := ExcessiveAddendumFinding(e)

	if f.Title != "Contrato 7/2015 (FMS): aditivos somam 46,37% de acréscimo, acima do limite de 25%" ||
		f.Detail != "Primeiro termo aditivo, 08/09/2015: 46,37%." || len(f.ActIDs) != 1 {
		t.Errorf("caso inesperado: %+v", f)
	}
}

func TestRenewedEmergencyFinding(t *testing.T) {
	r := RenewedEmergency{SupplierLabel: "Lógica Tecnologia Ltda", Organ: "PGM", Contracts: []EmergencyContract{
		{First: civilDate(2021, 1, 22), ActIDs: []string{"a"}},
		{First: civilDate(2022, 1, 21), ActIDs: []string{"b", "c"}},
		{First: civilDate(2022, 7, 28), ActIDs: []string{"d"}},
	}}

	f := RenewedEmergencyFinding(r)

	if f.Title != "Lógica Tecnologia Ltda em PGM: 3 contratações emergenciais seguidas, de 22/01/2021 a 28/07/2022" ||
		f.Detail != "Contratações emergenciais em 22/01/2021, 21/01/2022 e 28/07/2022." || len(f.ActIDs) != 4 {
		t.Errorf("caso inesperado: %+v", f)
	}
}
