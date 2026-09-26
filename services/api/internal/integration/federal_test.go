//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

type federalAPI map[string]map[string]string

func (f federalAPI) LatestMonth(context.Context, string) (time.Time, error) {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil
}

func (f federalAPI) ZipCSVs(_ context.Context, file string, each func(name string, header, row []string) error) (string, error) {
	for name, text := range f[file] {
		r := csv.NewReader(strings.NewReader(text))
		r.Comma = ';'
		rows, err := r.ReadAll()
		if err != nil {
			return "", err
		}
		for _, row := range rows[1:] {
			if err := each(name, rows[0], row); err != nil {
				return "", err
			}
		}
	}
	return "sha", nil
}

func federalFixture() federalAPI {
	return federalAPI{
		"emendas-parlamentares/UNICO": {
			"EmendasParlamentares.csv": "Código da Emenda;Ano da Emenda;Tipo de Emenda;Nome do Autor da Emenda;Número da emenda;Código Município IBGE;Nome Função;Nome Subfunção;Nome Programa;Nome Ação;Valor Empenhado;Valor Liquidado;Valor Pago\n" +
				"1;2024;Individual;PARLAMENTAR A;0001;3304904;Saúde;S;P;INCREMENTO;10,00;5,00;5,00\n",
			"EmendasParlamentares_PorFavorecido.csv": "Código da Emenda;Nome do Autor da Emenda;Tipo de Emenda;Ano/Mês;Código do Favorecido;Favorecido;Natureza Jurídica;Tipo Favorecido;UF Favorecido;Município Favorecido;Valor Recebido\n" +
				"1;PARLAMENTAR A;Individual;202601;" + fpVieira + ";F.P. VIEIRA;Sociedade;Pessoa Jurídica;RJ;SÃO GONÇALO;100,00\n",
		},
		"transferencias/202601": {"t.csv": "ANO / MÊS;TIPO TRANSFERÊNCIA;UF;NOME MUNICÍPIO;NOME ÓRGÃO;NOME FUNÇÃO;NOME PROGRAMA;NOME AÇÃO;LINGUAGEM CIDADÃ;CÓDIGO FAVORECIDO;NOME FAVORECIDO;VALOR TRANSFERIDO\n" +
			"202601;Legais;RJ;SAO GONCALO;Saúde;Saúde;P;A;;28636579000100;MUNICIPIO;1000,00\n" +
			"202601;Constitucionais;RJ;SAO GONCALO;Fazenda;Encargos especiais;P;FPM;;28636579000100;MUNICIPIO;50,00\n"},
	}
}

func loadFederal(t *testing.T, db *sql.DB) {
	t.Helper()
	now := func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }
	uc := usecase.NewLoadFederal(federalFixture(), postgres.NewFederalRepo(db), postgres.NewFetchRunRepo(db), discardObjects{}, now)
	if _, err := uc.Amendments(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Transfers(context.Background(), time.Time{}, time.Time{}); err != nil {
		t.Fatal(err)
	}
}

func TestFederalLoadStoresAndLinksTheFavored(t *testing.T) {
	_, db := newServerFor(t, semedHomologacao2025)
	repo := postgres.NewFederalRepo(db)

	loadFederal(t, db)
	loadFederal(t, db)

	amendments, err := repo.Amendments(context.Background())
	if err != nil || len(amendments) != 1 || amendments[0].PaidCents != 500 {
		t.Fatalf("emendas: %+v %v", amendments, err)
	}
	payments, err := repo.AmendmentPaymentsByCNPJ(context.Background(), fpVieira)
	if err != nil || len(payments) != 1 || payments[0].ValueCents != 10000 {
		t.Fatalf("pagamentos: %+v %v", payments, err)
	}
	totals, err := repo.TransferTotals(context.Background())
	if err != nil || len(totals) != 2 || totals[0].ValueCents != 100000 || totals[0].Year != 2026 {
		t.Fatalf("transferências: %+v %v", totals, err)
	}
	var links int
	if err := db.QueryRow(`SELECT count(*) FROM entity_links WHERE source = $1`, domain.SourceAmendments).Scan(&links); err != nil || links != 1 {
		t.Fatalf("ligações: %d %v", links, err)
	}
}

func TestFederalRouteAndCompanyPage(t *testing.T) {
	srv, db := newServerFor(t, semedHomologacao2025)
	loadFederal(t, db)

	var f struct {
		Transfers []struct {
			Year       int   `json:"year"`
			ValueCents int64 `json:"value_cents"`
		} `json:"transfers"`
		Amendments []struct {
			Author string `json:"author"`
		} `json:"amendments"`
		Favored []struct {
			CNPJ    string   `json:"cnpj"`
			Authors []string `json:"authors"`
			First   string   `json:"first"`
		} `json:"favored"`
	}
	getJSON(t, srv.URL+"/v1/federal", &f)
	if len(f.Transfers) != 2 || len(f.Amendments) != 1 || len(f.Favored) != 1 || f.Favored[0].CNPJ != fpVieira || f.Favored[0].First != "2026-01" {
		t.Fatalf("federal: %+v", f)
	}

	var company struct {
		Payments []struct {
			Author     string `json:"author"`
			ValueCents int64  `json:"value_cents"`
		} `json:"amendment_payments"`
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/"+fpVieira, &company)
	if len(company.Payments) != 1 || company.Payments[0].Author != "PARLAMENTAR A" {
		t.Fatalf("empresa: %+v", company.Payments)
	}
}
