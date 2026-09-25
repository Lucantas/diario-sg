package domain

import (
	"testing"
	"time"
)

const (
	pgmRatificacao2022 = "RECONHEÇO E RATIFICO a Dispensa Emergencial, Processo n.º 2360/2022 fundamento no art. 24, inciso IV da Lei n.º 8.666/93, para contratação de locação de equipamentos de informática."
	pgmContrato2022    = "EXTRATO DE CONTRATO Espécie: Emergencial: Base Legal: art. 24, inc. IV, da Lei n.º 8666/93. Processo: Processo n.º 2360/2022 Partes: Procuradoria Geral do Município de São Gonçalo X Lógica Tecnologia Ltda Objeto: Locação de Equipamentos de Informática Valor: R$ 91.020,00"
	pgmContrato2021    = "EXTRATO DE CONTRATO: Espécie: Emergencial. Base Legal: art. 24, inc. IV, da Lei n.º 8.666/93 Processo n.º 2042/21 Partes: Prefeitura Municipal de São Gonçalo – Procuradoria Geral e Secretaria de Fazenda x LOGICA TECNOLOGIA EIRELI. Objeto: locação de equipamentos de informática"
	pgmContratoJul2022 = "EXTRATO DE CONTRATO Espécie: Emergencial: Base Legal: art. 24, inc. IV, da Lei nº 8666/93. Processo: Processo nº 34550/2022 Partes: Procuradoria Geral do Município de São Gonçalo X Lógica Tecnologia Ltda Objeto: Locação de Equipamentos de Informática Valor: R$ 120.000,00"
)

func emergencyAct(id, organ string, day time.Time, body string, refs ...string) EmergencyAct {
	return EmergencyAct{ActID: id, Organ: organ, PublishedAt: day, Body: body, Refs: refs}
}

func TestFindRenewedEmergenciesFlagsTheSameSupplierAgainAndAgain(t *testing.T) {
	acts := []EmergencyAct{
		emergencyAct("2021", "PGM", civilDate(2021, 1, 22), pgmContrato2021, "processo:20422021"),
		emergencyAct("ratif", "", civilDate(2022, 1, 21), pgmRatificacao2022, "processo:23602022"),
		emergencyAct("jan", "PGM", civilDate(2022, 1, 21), pgmContrato2022, "processo:23602022"),
		emergencyAct("jul", "PGM", civilDate(2022, 7, 28), pgmContratoJul2022, "processo:345502022"),
	}

	got := FindRenewedEmergencies(acts)

	if len(got) != 1 || got[0].Supplier != "nome:logica tecnologia" || got[0].Organ != "PGM" || len(got[0].Contracts) != 3 {
		t.Fatalf("esperava 3 contratações seguidas da Lógica na PGM: %+v", got)
	}
	if ids := got[0].Contracts[1].ActIDs; len(ids) != 2 {
		t.Errorf("a ratificação deveria entrar na contratação de janeiro de 2022 pelo processo: %+v", got[0].Contracts[1])
	}
}

func TestFindRenewedEmergenciesIgnoresSimultaneousContracts(t *testing.T) {
	body := "EXTRATO DE CONTRATO Espécie: Emergencial. Base Legal: art. 24, inc. IV Partes: Secretaria Municipal de Assistência Social x DBNOVA Tecnologia Ltda. Objeto: locação de sistema"
	var acts []EmergencyAct
	for i, p := range []string{"66032021", "66042021", "66052021"} {
		acts = append(acts, emergencyAct(p, "SEMAS", civilDate(2021, 2, 26+i%2), body, "processo:"+p))
	}

	if got := FindRenewedEmergencies(acts); len(got) != 0 {
		t.Fatalf("contratações no mesmo mês não são renovação: %+v", got)
	}
}

func TestFindRenewedEmergenciesJoinsNameAndCNPJOfTheSameSupplier(t *testing.T) {
	withCNPJ := emergencyAct("a", "FMS", civilDate(2018, 8, 9), "EXTRATO DO CONTRATO FMS N.º 004/2018 PARTES: FUNDAÇÃO MUNICIPAL DE SAÚDE e CENTRO FLUMINENSE DE OXIGENOTERAPIA HIPERBÁRICA LTDA, CNPJ 08.116.346/0001-61. art. 24, inciso IV", "contrato:4/2018")
	withCNPJ.CNPJs = []string{"08116346000161", "39260120000163"}
	nameOnly := emergencyAct("b", "FMSSG", civilDate(2018, 11, 12), "RATIFICO com fundamento no art. 24, inciso IV, da Lei 8.666/93 em favor da empresa Centro Fluminense de Oxigenoterapia Hiperbárica Ltda, para tratamento", "processo:12902018")

	got := FindRenewedEmergencies([]EmergencyAct{withCNPJ, nameOnly})

	if len(got) != 1 || got[0].Supplier != "cnpj:08116346000161" || got[0].Organ != "FMS" || len(got[0].Contracts) != 2 {
		t.Fatalf("nome e CNPJ da mesma empresa deveriam se juntar: %+v", got)
	}
}

func TestFindRenewedEmergenciesStartsOverAfterTwoYears(t *testing.T) {
	acts := []EmergencyAct{
		emergencyAct("2013", "SEMIURB", civilDate(2013, 7, 10), "em favor da empresa Construtora Marquise S/A, art. 24, IV", "processo:1"),
		emergencyAct("2016", "SEMIURB", civilDate(2016, 7, 10), "em favor da empresa Construtora Marquise S/A, art. 24, IV", "processo:2"),
	}

	if got := FindRenewedEmergencies(acts); len(got) != 0 {
		t.Fatalf("contratações três anos depois não são renovação seguida: %+v", got)
	}
}

func TestFindRenewedEmergenciesJoinsNameReadFromAnActWithTheCNPJ(t *testing.T) {
	withCNPJ := emergencyAct("a", "FMS", civilDate(2018, 8, 9), "Contratado: CENTRO FLUMINENSE DE OXIGENOTERAPIA HIPERBÁRICA LTDA. Objeto: tratamento. art. 24, inciso IV", "contrato:4/2018")
	withCNPJ.CNPJs = []string{"08116346000161"}
	nameOnly := emergencyAct("b", "FMS", civilDate(2019, 9, 9), "RATIFICO com fundamento no art. 24, inciso IV em favor da empresa Centro Fluminense de Oxigenoterapia Hiperbárica Ltda, para tratamento", "processo:31672019")

	got := FindRenewedEmergencies([]EmergencyAct{withCNPJ, nameOnly})

	if len(got) != 1 || got[0].Supplier != "cnpj:08116346000161" || len(got[0].Contracts) != 2 {
		t.Fatalf("o nome lido do ato com CNPJ deveria juntar as duas: %+v", got)
	}
}

func TestFindRenewedEmergenciesKeepsTheSameNumberInDifferentOrgansApart(t *testing.T) {
	alfa := "EXTRATO DE CONTRATO Espécie: Emergencial. Partes: Secretaria Municipal de Assistência Social x Alfa Serviços Ltda. Objeto: limpeza"
	beta := "EXTRATO DE CONTRATO Espécie: Emergencial. Partes: Fundação Municipal de Saúde x Beta Serviços Ltda. Objeto: limpeza"
	acts := []EmergencyAct{
		emergencyAct("alfa-1", "SEMAS", civilDate(2022, 1, 10), alfa, "contrato:1/2022"),
		emergencyAct("beta-1", "FMS", civilDate(2022, 1, 12), beta, "contrato:1/2022"),
		emergencyAct("alfa-2", "SEMAS", civilDate(2022, 7, 10), alfa, "contrato:9/2022"),
	}

	got := FindRenewedEmergencies(acts)

	if len(got) != 1 || got[0].Supplier != "nome:alfa servicos" || got[0].Organ != "SEMAS" || len(got[0].Contracts) != 2 {
		t.Fatalf("o contrato 1/2022 da FMS não é o da SEMAS: %+v", got)
	}
}

func TestFindRenewedEmergenciesSkipsContractsWithoutOrgan(t *testing.T) {
	body := "em favor da empresa Gama Serviços Ltda, art. 24, IV"
	acts := []EmergencyAct{
		emergencyAct("1", "", civilDate(2022, 1, 10), body, "processo:1"),
		emergencyAct("2", "", civilDate(2022, 7, 10), body, "processo:2"),
	}

	if got := FindRenewedEmergencies(acts); len(got) != 0 {
		t.Fatalf("sem órgão não dá para dizer que é o mesmo órgão: %+v", got)
	}
}
