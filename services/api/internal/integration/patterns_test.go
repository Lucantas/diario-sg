//go:build integration

package integration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const (
	semmaDispensa2020 = `EXTRATO DE RATIFICAÇÃO DE DISPENSA LICITAÇÃO
PROCESSO ADMINSTRATIVO Nº 21577/19
RATIFICO a situação de dispensa de licitação: Processo nº: 21.577/2019 Objeto: Aquisição de (02) dois decibelímetros digitais.
Favorecida: INSTRUTHERM INSTRUMENTOS DE MEDIÇÃO LTDA, CNPJ 53.775.862/0001-52. Valor Global: R$ 11.862,00 (onze mil oitocentos e sessenta e dois reais).
Legislação Aplicável: O presente Termo tem por fundamento legal o art. 24, inciso II c/c 23, II da Lei nº 8.666/93.`
	semadDispensa2020 = `EXTRATO DE RATIFICAÇÃO DE DISPENSA DE LICITAÇÃO
Consubstanciado no parecer jurídico exarado pela Procuradoria Geral do Município de São Gonçalo, RATIFICO a situação de Dispensa de Licitação, baseada no art. 24, inciso II da
Lei Federal nº. 8.666 de 21 de junho de 1993, que tem por objeto
a aquisição de Equipamentos de Avaliação Ambiental, em favor da empresa INSTRUTHERM INSTRUMENTOS DE MEDIÇÃO LTDA., inscrita no
CNPJ 53.775.862/0001-52, Processo nº 53.639/2018, no valor de
R$ 17.049,60 (dezessete mil quarenta e nove reais e sessenta
centavos).`
	semadRepublicada2020 = `EXTRATO DE RATIFICAÇÃO DE DISPENSA DE LICITAÇÃO
Consubstanciado no parecer jurídico exarado pela Procuradoria Geral do Município de São Gonçalo, RATIFICO a situação de Dispensa de Licitação, baseada no art. 24, inciso II da
Lei Federal nº 8.666 de 21 de junho de 1993, que tem por objeto
a aquisição de Equipamentos de Avaliação Ambiental, em favor da empresa INSTRUTHERM INSTRUMENTOS DE MEDIÇÃO LTDA., inscrita no
CNPJ 53.775.862/0001-52, Processo nº 56.639/2018, no valor de
R$ 17.049,60 (Dezessete mil quarenta e nove reais e sessenta
centavos).
Republicado por incorreção da PMSG.`
	semtranRatificacao2024 = `EXTRATO DE RATIFICAÇÃO DE DISPENSA DE LICITAÇÃO.
Processo Administrativo n.º 18.635/2024.
PARTES: SECRETARIA MUNICIPAL DE TRANSPORTES e LM
CURSOS DE TRANSITO SOCIEDADE UNIPESSOAL LTDA - CNPJ nº:
18.657.198/0001-46.
RATIFICO a DISPENSA DE
LICITAÇÃO, conforme artigo 75, inciso II da Lei de Licitações n.º
14.133 de 1º de abril de 2021.
VALOR TOTAL: R$ 57.999,60 (cinquenta e sete mil novecentos e
noventa e nove reais e sessenta centavos)`
	semtranTermo2024 = `EXTRATO DO TERMO DE DISPENSA DE LICITAÇÃO.
Processo Administrativo n.º 18.635/2024.
Dispensa de licitação, conforme artigo 75, inciso II, da Lei de
Licitações n.º 14.133/2021. HOMOLOGO a DISPENSA DE LICITAÇÃO, em favor da empresa LM
CURSOS DE TRANSITO SOCIEDADE UNIPESSOAL LTDA - CNPJ nº:
18.657.198/0001-46, no valor total de R$ 57.999,60.`
	semtranContrato2024 = `EXTRATO DO CONTRATO N.º 02/SEMTRAN/24
Processo Administrativo N.º 18.635/2024
Partes: Secretaria Municipal de Transportes e LM Cursos de
Transito Sociedade Unipessoal LTDA - CNPJ nº: 18.657.198/0001-46.
Licitação: Dispensa de Licitação nº 001/2024.
Valor Global: R$ 57.999,60 (cinquenta e sete mil novecentos e
noventa e nove reais e sessenta centavos)
Fundamento Legal: Artigo 75, inciso II da Lei de Licitações n.º 14.133
de 1º de abril de 2021.`
)

type patternsResponse struct {
	Items []struct {
		ID       string `json:"id"`
		Rule     string `json:"rule"`
		Findings []struct {
			Title string `json:"title"`
			Acts  []struct {
				ID string `json:"id"`
			} `json:"acts"`
			Search *struct {
				Type string `json:"type"`
			} `json:"search"`
		} `json:"findings"`
	} `json:"items"`
}

func TestPatternsFlagSplitDispensasAndIgnoreRepublishedProcesses(t *testing.T) {
	srv, db := newServerFor(t, semmaDispensa2020)
	setOnlyGazetteDate(t, db, "2020-02-19")
	indexAt(t, db, time.Date(2020, 5, 22, 0, 0, 0, 0, time.UTC), semadDispensa2020)
	indexAt(t, db, time.Date(2020, 5, 28, 0, 0, 0, 0, time.UTC), semadRepublicada2020)
	indexAt(t, db, time.Date(2024, 10, 3, 0, 0, 0, 0, time.UTC), semtranRatificacao2024)
	indexAt(t, db, time.Date(2024, 10, 4, 0, 0, 0, 0, time.UTC), semtranTermo2024)
	indexAt(t, db, time.Date(2024, 10, 9, 0, 0, 0, 0, time.UTC), semtranContrato2024)

	var res patternsResponse
	getJSON(t, srv.URL+"/v1/patterns", &res)

	if len(res.Items) != 9 || res.Items[0].ID != "fracionamento_dispensa" || res.Items[1].ID != "aditivo_acima_do_limite" ||
		res.Items[2].ID != "emergencial_renovada" || res.Items[3].ID != "pico_pessoal_eleicao" {
		t.Fatalf("padrões inesperados: %+v", res)
	}
	split := res.Items[0].Findings
	if len(split) != 1 || !strings.HasPrefix(split[0].Title, "CNPJ 53.775.862/0001-52 em 2020: 2 dispensas de compras e serviços somam R$ 28.911,60") ||
		len(split[0].Acts) != 3 || split[0].Search != nil {
		t.Fatalf("fracionamento inesperado: %+v", split)
	}
	for _, f := range split {
		if strings.Contains(f.Title, "18.657.198/0001-46") {
			t.Fatalf("o mesmo processo publicado três vezes não é fracionamento: %+v", f)
		}
	}
	if res.Items[3].Findings == nil || len(res.Items[3].Findings) != 0 {
		t.Fatalf("sem nomeações não há pico: %+v", res.Items[3])
	}
}

func TestMonthlyActCountsByTypeAndMonth(t *testing.T) {
	_, db := newServerFor(t, "PORTARIA Nº 1/2019\nO PREFEITO resolve NOMEAR Fulana de Tal para Assessora.\nPORTARIA Nº 2/2019\nResolve EXONERAR Beltrano do cargo de Diretor.")
	setOnlyGazetteDate(t, db, "2019-08-05")
	indexAt(t, db, time.Date(2019, 8, 20, 0, 0, 0, 0, time.UTC), "PORTARIA Nº 3/2019\nO PREFEITO resolve NOMEAR Sicrana para Assessora.")

	counts, err := postgres.NewPatternRepo(db).MonthlyActCounts(context.Background(), nil, "diario_prefeitura")
	if err != nil || len(counts) != 0 {
		t.Fatalf("sem tipos não há contagem: %+v %v", counts, err)
	}
	counts, err = postgres.NewPatternRepo(db).MonthlyActCounts(context.Background(), []domain.ActType{domain.ActNomeacao, domain.ActExoneracao}, "diario_prefeitura")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, c := range counts {
		got[string(c.Type)+"-"+c.Month.String()] = c.Count
	}
	if got["nomeacao-August"] != 2 || got["exoneracao-August"] != 1 || len(counts) != 2 {
		t.Fatalf("contagens inesperadas: %+v", counts)
	}
}

func indexAt(t *testing.T, db *sql.DB, day time.Time, text string) {
	t.Helper()
	sum := sha256.Sum256([]byte(day.String() + text))
	checksum := hex.EncodeToString(sum[:])
	in := usecase.IndexGazetteInput{EditionNumber: day.Format("20060102"), PublishedAt: day,
		SourceURL: "https://exemplo/" + day.Format("2006_01_02") + ".pdf", StoragePath: checksum + ".pdf", Checksum: checksum}
	idx := usecase.NewIndexGazette(postgres.NewGazetteRepo(db), stringStore(text), passthroughExtractor{}, parser.Set{}, entities.New(), &recPub{})
	if err := idx.Execute(context.Background(), in); err != nil {
		t.Fatal(err)
	}
}

func setOnlyGazetteDate(t *testing.T, db *sql.DB, day string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE gazettes SET published_at = $1`, day); err != nil {
		t.Fatal(err)
	}
}

const (
	fmsAditivo2015 = `EXTRATO DE TERMO ADITIVO DE CONTRATO
PROCEDIMENTO ADMINISTRATIVO N.º 0767/2014 CONTRATO FMS N.° 007/2015
PARTES: FUNDAÇÃO MUNICIPAL DE SAÚDE DE SÃO GONÇALO, e COMERCIAL DE EQUIPAMENTOS CNL DE SÃO GONÇALO LTDA ME,
inscrita no CNPJ/MF sob o n.º 13.391.199/0001-78.
OBJETO: O presente Termo Aditivo tem por objetivo o acréscimo equivalente a 46,37% do valor inicial contratado, o que perfaz
um total de R$ 240.802,53 (duzentos e quarenta mil, oitocentos e dois Reais e cinquenta e três Centavos).`
	pgmEmergencial2022 = `EXTRATO DE CONTRATO
Espécie: Emergencial: Base Legal: art. 24, inc. IV, da Lei n.º 8666/93. Processo: Processo n.º 2360/2022 Partes: Procuradoria
Geral do Município de São Gonçalo X Lógica Tecnologia Ltda Objeto: Locação de Equipamentos de Informática Valor: R$ 91.020,00
(noventa e um mil, vinte reais)`
)

func TestIndexingStoresLegalBasisAndDeclaredIncrease(t *testing.T) {
	_, db := newServerFor(t, fmsAditivo2015)
	indexAt(t, db, time.Date(2022, 1, 21, 0, 0, 0, 0, time.UTC), pgmEmergencial2022)

	rows, err := db.Query(`SELECT a.type, array_to_string(a.legal_basis, ','), a.declared_increase_bp FROM acts a ORDER BY a.declared_increase_bp DESC`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var typ, basis string
		var bp int
		if err := rows.Scan(&typ, &basis, &bp); err != nil {
			t.Fatal(err)
		}
		got = append(got, fmt.Sprintf("%s|%s|%d", typ, basis, bp))
	}
	if strings.Join(got, " ") != "aditivo||4637 contrato|art24:IV|0" {
		t.Fatalf("fatos gravados inesperados: %v", got)
	}
}

const (
	pgmEmergencial2021 = `EXTRATO DE CONTRATO:
Espécie: Emergencial. Base Legal: art. 24, inc. IV, da Lei n.º
8.666/93
Processo n.º 2042/21
Partes: Prefeitura Municipal de São Gonçalo – Procuradoria
Geral x LOGICA TECNOLOGIA EIRELI - EPP.
Objeto: Locação de Equipamentos de Informática com
suprimentos
Valor Mensal R$ 11.710,00; Prazo: 06 (seis) meses.`
	pgmRatificacao2022 = `TERMO DE RATIFICAÇÃO EMERGENCIA
RECONHEÇO E RATIFICO com base no Art. 26 da Lei Federal n.º 8.666/93, e a vista do Parecer da Procuradoria-Geral do Município,
a Dispensa Emergencial, Processo n.º 2360/2022 fundamento no art. 24, inciso IV da Lei n.º 8.666/93, para contratação de locação
de equipamentos de informática.`
	pgmEmergencialJul2022 = `EXTRATO DE CONTRATO
Espécie: Emergencial: Base Legal: art. 24, inc. IV, da Lei nº 8666/93. Processo: Processo nº 34550/2022 Partes: Procuradoria
Geral do Município de São Gonçalo X Lógica Tecnologia Ltda Objeto: Locação de Equipamentos de Informática Valor: R$ 120.000,00
(cento e vinte mil reais)`
)

func TestPatternsFlagExcessiveAddendaAndRenewedEmergencies(t *testing.T) {
	srv, db := newServerFor(t, "ATOS DO PREFEITO\nFMS\n"+fmsAditivo2015)
	setOnlyGazetteDate(t, db, "2015-09-08")
	indexAt(t, db, time.Date(2021, 1, 22, 0, 0, 0, 0, time.UTC), "ATOS DO PREFEITO\nPGM\n"+pgmEmergencial2021)
	indexAt(t, db, time.Date(2022, 1, 20, 0, 0, 0, 0, time.UTC), pgmRatificacao2022)
	indexAt(t, db, time.Date(2022, 1, 21, 0, 0, 0, 0, time.UTC), "ATOS DO PREFEITO\nPGM\n"+pgmEmergencial2022)
	indexAt(t, db, time.Date(2022, 7, 28, 0, 0, 0, 0, time.UTC), "ATOS DO PREFEITO\nPGM\n"+pgmEmergencialJul2022)

	var res patternsResponse
	getJSON(t, srv.URL+"/v1/patterns", &res)

	addenda := res.Items[1].Findings
	if len(addenda) != 1 || addenda[0].Title != "Contrato 7/2015 (FMS): aditivos somam 46,37% de acréscimo, acima do limite de 25%" || len(addenda[0].Acts) != 1 {
		t.Fatalf("aditivo inesperado: %+v", addenda)
	}
	renewed := res.Items[2].Findings
	if len(renewed) != 1 || renewed[0].Title != "LOGICA TECNOLOGIA EIRELI em PGM: 3 contratações emergenciais seguidas, de 22/01/2021 a 28/07/2022" ||
		len(renewed[0].Acts) != 4 {
		t.Fatalf("emergencial inesperada: %+v", renewed)
	}
}
