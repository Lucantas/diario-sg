package domain

import (
	"strings"
	"testing"
	"time"
)

const prefeituraPayJSON = `{"success":true,"data":[
{"cpf":"***.867.667-**","chapa":"1302**","nome":"ALEXANDRE COUTINHO DE SA","cargo":"SECRETARIO MUNICIPAL","funcao":"SECRETARIO MUNICIPAL","organograma":"78 - SECRETARIA MUNICIPAL DE COMUNICACAO SOCIAL","local":"78.01","referencia":"SI12","letra":"SM","remuneracao":16754.64,"jornada":"240 horas mensais"},
{"cpf":"***.111.222-**","chapa":"1316**","nome":"DANIEL LIMA","cargo":"AGENTE DE SAUDE AMBIENTAL","funcao":"SECRETARIO MUNICIPAL","organograma":"20 - SECRETARIA MUNICIPAL DE SAUDE","local":"20.01","referencia":"SI12","letra":"SM","remuneracao":16754.64,"jornada":""},
{"cpf":"***.111.333-**","chapa":"1400**","nome":"NELSON RUAS DOS SANTOS","cargo":"PREFEITO","funcao":"PREFEITO","organograma":"01 - GABINETE DO PREFEITO","local":"01","referencia":"SI12","letra":"PM","remuneracao":23813.22,"jornada":""},
{"cpf":"***.111.444-**","chapa":"1500**","nome":"JANUZA BRANDAO ASSAD SANTOS","cargo":"PROCURADOR GERAL DO MUNICIPIO","funcao":"PROCURADOR GERAL DO MUNICIPIO","organograma":"05 - PROCURADORIA GERAL DO MUNICIPIO","local":"05","referencia":"SI12","letra":"SM","remuneracao":16754.64,"jornada":""},
{"cpf":"***.111.555-**","chapa":"1600**","nome":"MARIA ESCOLAR","cargo":"PROFESSOR","funcao":"SECRETARIO ESCOLAR","organograma":"30 - SECRETARIA MUNICIPAL DE EDUCACAO","local":"30","referencia":"P1","letra":"A","remuneracao":4000.00,"jornada":""},
{"cpf":"***.111.666-**","chapa":"1700**","nome":"JOSE SUB","cargo":"SUBSECRETARIO","funcao":"SUBSECRETARIO","organograma":"30 - SECRETARIA MUNICIPAL DE EDUCACAO","local":"30","referencia":"SI11","letra":"SS","remuneracao":12000.00,"jornada":""}
]}`

const camaraPayJSON = `[
{"ano":"2026","mes":"08","nome":"CLAUDIO LUIZ ABREU DA SILVA","documento":"***..11.1.1-**","matricula":"190700","cargo":"VEREADOR","regime":"Agente Político","secretaria":"VEREADOR CACAU","valor_padrao":21840.43,"nome_rem01":"Vencimentos","valor_rem01":21840.43,"nome_rem02":"Descontos","valor_rem02":5813.74,"nome_rem03":"Líquido","valor_rem03":16026.69,"nome_rem04":null,"valor_rem04":0},
{"ano":"2026","mes":"08","nome":"PATRICIA HELENA DA SILVA RODRIGUES ","documento":"***","matricula":"190701","cargo":"VEREADOR","regime":"Agente Político","secretaria":"VEREADORA PATRICIA SILVA","valor_padrao":21840.43,"nome_rem01":"Vencimentos","valor_rem01":21840.43,"nome_rem02":"Descontos","valor_rem02":5813.74,"nome_rem03":"Líquido","valor_rem03":16026.69},
{"ano":"2026","mes":"08","nome":"FULANO ASSESSOR","documento":"***","matricula":"200000","cargo":"ASSESSOR PARLAMENTAR","regime":"Comissionado","secretaria":"VEREADOR CACAU","valor_padrao":3000,"nome_rem01":"Vencimentos","valor_rem01":3000,"nome_rem02":"Descontos","valor_rem02":300,"nome_rem03":"Líquido","valor_rem03":2700}
]`

const councillorsJSON = `{"Parametros":[
{"Nome_Vereador":"CACAU","Nome":"CLAUDIO LUÍS ABREU DA SILVA","Partido":"MDB","Legislatura":4,"Situacao":"Ativo","MesaDiretora":false,"DataNascimento":"1970-01-01","Email":"x@y","DpTelefone":"21 0000"},
{"Nome_Vereador":"PATRÍCIA SILVA","Nome":"PATRÍCIA HELENA DA SILVA RODRIGUES","Partido":"PL","Legislatura":4,"Situacao":"Ativo","MesaDiretora":true}
],"Total":2}`

func TestParsePrefeituraPayKeepsOnlyPoliticalAgents(t *testing.T) {
	rows, err := ParsePrefeituraPay([]byte(prefeituraPayJSON), 2026, 8)

	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range rows {
		got = append(got, r.Role+":"+r.Name)
	}
	if strings.Join(got, ",") != "secretario:ALEXANDRE COUTINHO DE SA,secretario:DANIEL LIMA,prefeito:NELSON RUAS DOS SANTOS,procurador_geral:JANUZA BRANDAO ASSAD SANTOS" {
		t.Fatalf("agentes: %v", got)
	}
	first := rows[0]
	if first.Body != BodyPrefeitura || !first.Month.Equal(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)) || first.Office != "SECRETARIA MUNICIPAL DE COMUNICACAO SOCIAL" ||
		first.GrossCents != 1675464 || first.DiscountCents != nil || first.NetCents != nil || first.NameKey != "ALEXANDRE COUTINHO DE SA" {
		t.Errorf("linha: %+v", first)
	}
}

func TestParsePrefeituraPayWithoutDataIsEmpty(t *testing.T) {
	rows, err := ParsePrefeituraPay([]byte(`{"success":true,"data":[]}`), 2010, 1)

	if err != nil || len(rows) != 0 {
		t.Errorf("mês vazio: %v %v", rows, err)
	}
	if _, err := ParsePrefeituraPay([]byte(`{"success":false}`), 2010, 1); err == nil {
		t.Error("resposta sem sucesso aceita")
	}
}

func TestParseCamaraPayKeepsOnlyCouncillors(t *testing.T) {
	rows, err := ParseCamaraPay([]byte(camaraPayJSON))

	if err != nil || len(rows) != 2 {
		t.Fatalf("vereadores: %+v %v", rows, err)
	}
	r := rows[0]
	if r.Body != BodyCamara || r.Role != RoleVereador || r.Office != "VEREADOR CACAU" || r.GrossCents != 2184043 ||
		r.DiscountCents == nil || *r.DiscountCents != 581374 || r.NetCents == nil || *r.NetCents != 1602669 ||
		!r.Month.Equal(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("linha: %+v", r)
	}
	if rows[1].Name != "PATRICIA HELENA DA SILVA RODRIGUES" {
		t.Errorf("nome sem espaço final: %q", rows[1].Name)
	}
}

func TestParseCamaraPayWithoutDataIsEmpty(t *testing.T) {
	rows, err := ParseCamaraPay([]byte(`{"mensagem":"Informamos que, no período consultado, não houve informações disponibilizadas"}`))

	if err != nil || len(rows) != 0 {
		t.Errorf("ano vazio: %v %v", rows, err)
	}
}

func TestParseCouncillorsKeepsOnlyPublicMandateFields(t *testing.T) {
	rows, err := ParseCouncillors([]byte(councillorsJSON))

	if err != nil || len(rows) != 2 {
		t.Fatalf("vereadores: %+v %v", rows, err)
	}
	if rows[0] != (Councillor{Legislature: 4, Name: "CLAUDIO LUÍS ABREU DA SILVA", NameKey: "CLAUDIO LUIS ABREU DA SILVA", ParliamentaryName: "CACAU",
		Party: "MDB", Situation: "Ativo"}) {
		t.Errorf("vereador: %+v", rows[0])
	}
}

func TestBuildPoliticalAgentsGroupsMonthsAndJoinsTheCouncillor(t *testing.T) {
	camara, _ := ParseCamaraPay([]byte(camaraPayJSON))
	july := camara[0]
	july.Month = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	prefeitura, _ := ParsePrefeituraPay([]byte(prefeituraPayJSON), 2026, 8)
	councillors, _ := ParseCouncillors([]byte(councillorsJSON))

	agents := BuildPoliticalAgents(append(append(camara, july), prefeitura...), councillors)

	if len(agents) != 6 {
		t.Fatalf("agentes: %d %+v", len(agents), agents)
	}
	if agents[0].Role != RolePrefeito || agents[len(agents)-1].Role != RoleVereador {
		t.Errorf("ordem por papel: %v ... %v", agents[0].Role, agents[len(agents)-1].Role)
	}
	var cacau PoliticalAgent
	for _, a := range agents {
		if a.Name == "CLAUDIO LUIZ ABREU DA SILVA" {
			cacau = a
		}
	}
	if cacau.Party != "MDB" || cacau.ParliamentaryName != "CACAU" || len(cacau.Months) != 2 || !cacau.Months[0].Month.Before(cacau.Months[1].Month) {
		t.Errorf("vereador pelo nome parlamentar: %+v", cacau)
	}
}

func TestSubsidyNormsCoverEveryRole(t *testing.T) {
	roles := map[string]bool{}
	for _, n := range SubsidyNorms {
		roles[n.Role] = true
		if n.Cents <= 0 || n.Norm == "" || n.Search == "" {
			t.Errorf("norma incompleta: %+v", n)
		}
	}
	for _, r := range []string{RolePrefeito, RoleVicePrefeito, RoleSecretario, RoleProcuradorGeral, RoleVereador} {
		if !roles[r] {
			t.Errorf("sem norma para %s", r)
		}
	}
}

func TestMergeAgentPaySumsRepeatedLines(t *testing.T) {
	discount := int64(100)
	rows := []AgentPay{
		{Body: BodyPrefeitura, Month: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC), NameKey: "A", Role: RolePrefeito, Office: "G", GrossCents: 1000},
		{Body: BodyPrefeitura, Month: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC), NameKey: "A", Role: RolePrefeito, Office: "G", GrossCents: 500, DiscountCents: &discount},
		{Body: BodyPrefeitura, Month: time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC), NameKey: "A", Role: RolePrefeito, Office: "G", GrossCents: 1000},
	}

	merged := MergeAgentPay(rows)

	if len(merged) != 2 || merged[0].GrossCents != 1500 || merged[0].DiscountCents == nil || *merged[0].DiscountCents != 100 {
		t.Errorf("soma: %+v", merged)
	}
	if rows[0].GrossCents != 1000 {
		t.Error("alterou a entrada")
	}
}

func TestParseCamaraPayRejectsOtherMessages(t *testing.T) {
	if _, err := ParseCamaraPay([]byte(`{"mensagem":"Sistema em manutenção"}`)); err == nil {
		t.Error("mensagem de erro tratada como ano sem dados")
	}
}

func TestParseCamaraPayRejectsANonNumericValue(t *testing.T) {
	body := `[{"ano":"2026","mes":"08","nome":"X","cargo":"VEREADOR","regime":"Agente Político","secretaria":"VEREADOR X","nome_rem01":"Vencimentos","valor_rem01":"21840,43"}]`

	if _, err := ParseCamaraPay([]byte(body)); err == nil {
		t.Error("valor em texto virou zero")
	}
}
