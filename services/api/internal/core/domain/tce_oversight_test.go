package domain

import (
	"testing"
	"time"
)

func TestParseTCEAccountsKeepsSaoGoncalo(t *testing.T) {
	body := []byte(`[{"Municipio":"SÃO GONÇALO","Regiao":"Metropolitana","Ano":2024,"Indicador":"FAVORÁVEL","Processo":"213638-8/2025","Responsavel":"NOME DO PREFEITO"},` +
		`{"Municipio":"NITERÓI","Ano":2024,"Indicador":"FAVORÁVEL","Processo":"1/2025","Responsavel":"OUTRO"}]`)

	got, err := ParseTCEAccounts(body)

	if err != nil || len(got) != 1 || got[0].Year != 2024 || got[0].Opinion != "FAVORÁVEL" || got[0].Process != "213638-8/2025" {
		t.Fatalf("veio %+v %v", got, err)
	}
}

func TestParseTCEPenaltiesReadsValueAndSession(t *testing.T) {
	body := []byte(`[{"Processo":"200473-0/2015","AnoCondenacao":2022,"ValorPenalidade":42145.75,"Condenacao":"657/2022-2","Ente":"SAO GONCALO",` +
		`"NomeOrgao":"PREFEITURA SÃO GONÇALO","TipoEnte":"MUNICIPAL","GrupoNatureza":"PRESTAÇÃO DE CONTAS","DataSessao":"2022-05-25T00:00:00"},` +
		`{"Processo":"1/2020","Ente":"SAO JOAO DE MERITI","DataSessao":"2025-01-27T00:00:00"}]`)

	got, err := ParseTCEPenalties(body)

	if err != nil || len(got) != 1 {
		t.Fatalf("veio %+v %v", got, err)
	}
	p := got[0]
	if p.ValueCents != 4214575 || p.Condemnation != "657/2022-2" || p.SessionDate == nil || p.SessionDate.Format("02/01/2006") != "25/05/2022" {
		t.Fatalf("condenação: %+v", p)
	}
}

func TestParseStalledWorksNormalizesTheCNPJ(t *testing.T) {
	body := []byte(`{"Obras":[{"AnoParalisacao":2016,"DataParalisacao":"2016-08-01","Ente":"SAO GONCALO","Nome":"PREFEITURA SÃO GONÇALO",` +
		`"FuncaoGoverno":"URBANISMO","NumeroContrato":"1010446-30","CNPJContratada":"01992029000160","NomeContratada":"R.C VIEIRA ENGENHARIA LTDA",` +
		`"ValorTotalContrato":2290684.0,"ValorPagoObra":848815.24,"TempoParalizacao":"ACIMA DE 2 ANOS","MotivoParalisacao":"REPASSE DE CONVÊNIOS - ATRASO",` +
		`"DataInicioObra":"2015-07-01","StatusContrato":"VIGENTE"}],"Count":1}`)

	got, err := ParseStalledWorks(body)

	if err != nil || len(got) != 1 || got[0].CNPJ != "01992029000160" || got[0].PaidCents != 84881524 || got[0].StartedAt.Year() != 2015 {
		t.Fatalf("veio %+v %v", got, err)
	}
}

func TestTCEProcessSearchUsesTheDiarioSpelling(t *testing.T) {
	if got := TCEProcessSearch("214824-1/2014"); got != `"214.824" TCE` {
		t.Fatalf("busca: %s", got)
	}
	if got := TCEProcessSearch("12/2020"); got != "" {
		t.Fatalf("número curto: %s", got)
	}
}

func TestGroupPenaltiesSumsByProcessNewestFirst(t *testing.T) {
	older, newer := mustDay("2022-05-25"), mustDay("2025-04-07")
	penalties := []TCEPenalty{
		{Condemnation: "1-1", Process: "200473-0/2015", ValueCents: 100, Organ: "PREFEITURA", Nature: "PRESTAÇÃO DE CONTAS", SessionDate: older},
		{Condemnation: "2-0", Process: "200783-6/2020", ValueCents: 5, Organ: "PREFEITURA", Nature: "DENÚNCIA", SessionDate: newer},
		{Condemnation: "1-2", Process: "200473-0/2015", ValueCents: 50, Organ: "FMS", Nature: "PRESTAÇÃO DE CONTAS", SessionDate: older},
	}

	got := GroupPenalties(penalties)

	if len(got) != 2 || got[0].Process != "200783-6/2020" || got[1].TotalCents != 150 || len(got[1].Condemnations) != 2 ||
		len(got[1].Organs) != 2 || len(got[1].Natures) != 1 || got[1].Search != `"200.473" TCE` {
		t.Fatalf("processos: %+v", got)
	}
}

func mustDay(s string) *time.Time {
	d, err := tceDate(s)
	if err != nil {
		panic(err)
	}
	return d
}
