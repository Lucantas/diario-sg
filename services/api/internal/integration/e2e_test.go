//go:build integration

// Teste de integração: Postgres real + camada HTTP + casos de uso.
// Rode com: TEST_DATABASE_URL=postgres://... go test -tags integration ./internal/integration/
// No CI, o GitHub Actions sobe um Postgres como service container.
package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/email"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
	httpapi "github.com/seu-usuario/diario-sg/services/api/internal/presentation/http"
	"github.com/seu-usuario/diario-sg/services/api/migrations"
)

const gazetteText = `DECRETO Nº 100/2026
Dispõe sobre o horário das repartições.
PORTARIA Nº 2.345/2026
O PREFEITO resolve NOMEAR Fulana de Tal para Assessora da Secretaria de Saúde.
EXTRATO DO CONTRATO Nº 55/2026
Contratada: Empresa Exemplo LTDA, CNPJ: 12.345.678/0001-90. Objeto: fornecimento de medicamentos. Valor: R$ 120.000,00.
Contrato nº 55/2026. Processo Administrativo nº 8.189/2025.
PORTARIA Nº 2.346/2026
Resolve EXONERAR JOSÉ DA SILVA do cargo de Diretor de Conservação.
EDITAL DE CONVOCAÇÃO 01/2026
Convocados: JOSÉ CARLOS MOURA, MARIA SILVA PRADO, JOSÉ AUGUSTO LEAL, ANA SILVA MOTA, JOSÉ PEDRO NUNES, RITA SILVA REIS.`

type textStore struct{}

func (textStore) Get(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(gazetteText)), nil
}

type passthroughExtractor struct{}

func (passthroughExtractor) Extract(_ context.Context, r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	return string(b), err
}

type recPub struct{ ids []string }

func (p *recPub) GazetteIndexed(_ context.Context, id string, _ int) error {
	p.ids = append(p.ids, id)
	return nil
}

type inbox struct{ msgs []email.Message }

func (i *inbox) Send(_ context.Context, m email.Message) error {
	i.msgs = append(i.msgs, m)
	return nil
}

func TestEndToEnd(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	ctx := context.Background()
	db := openTestDB(t, ctx, url)
	defer db.Close()
	if _, err := postgres.Migrate(ctx, db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE gazettes, subscriptions CASCADE`); err != nil {
		t.Fatal(err)
	}

	gaz, acts := postgres.NewGazetteRepo(db), postgres.NewActRepo(db)
	subs, nlog := postgres.NewSubscriptionRepo(db), postgres.NewNotificationLog(db)
	pub := &recPub{}
	box := &inbox{}
	notifier := email.NewNotifier(box, "https://web.exemplo")

	// 1. Indexação (duas vezes: tem que ser idempotente)
	idx := usecase.NewIndexGazette(gaz, textStore{}, passthroughExtractor{}, parser.New(), entities.New(), pub)
	in := usecase.IndexGazetteInput{EditionNumber: "1", PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
		SourceURL: "https://exemplo/1.pdf", StoragePath: "1.pdf", Checksum: strings.Repeat("b", 64)}
	for i := 0; i < 2; i++ {
		if err := idx.Execute(ctx, in); err != nil {
			t.Fatal(err)
		}
	}

	api := &httpapi.API{Search: usecase.NewSearchActs(acts), Gazette: usecase.NewGetGazette(gaz, acts),
		Company: usecase.NewGetCompany(acts), Stats: usecase.NewActStats(acts),
		Subscriptions: usecase.NewSubscriptions(subs, notifier), Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	srv := httptest.NewServer(api.Routes())
	defer srv.Close()

	// 2. Busca textual em português (stemming: "medicamento" acha "medicamentos")
	var res struct {
		Items []struct {
			Type, Snippet string
			CNPJs         []string `json:"cnpjs"`
		} `json:"items"`
		Total int `json:"total"`
	}
	getJSON(t, srv.URL+"/v1/acts?q=medicamento", &res)
	if res.Total != 1 || res.Items[0].Type != "contrato" || !strings.Contains(res.Items[0].Snippet, "⟦") {
		t.Fatalf("busca inesperada: %+v", res)
	}
	if len(res.Items[0].CNPJs) != 1 || res.Items[0].CNPJs[0] != "12345678000190" {
		t.Fatalf("resultado deve listar os CNPJs do ato: %+v", res.Items[0])
	}

	// 2b. Nome sem acento encontra o nome acentuado em caixa alta, e a frase
	// exata vem antes da lista que repete "JOSÉ" e "SILVA" soltos
	getJSON(t, srv.URL+"/v1/acts?q=jose+da+silva", &res)
	if res.Total != 2 || res.Items[0].Type != "exoneracao" || res.Items[1].Type != "edital" || !strings.Contains(res.Items[0].Snippet, "⟦JOSÉ⟧") {
		t.Fatalf("busca sem acento inesperada: %+v", res)
	}

	// 2b'. Entre aspas só a frase exata casa
	getJSON(t, srv.URL+"/v1/acts?q=%22jose+da+silva%22", &res)
	if res.Total != 1 || res.Items[0].Type != "exoneracao" || !strings.Contains(res.Items[0].Snippet, "⟦JOSÉ⟧") {
		t.Fatalf("busca por frase inesperada: %+v", res)
	}

	// 2c. Substring que o tokenizador não trata (trecho de CNPJ) casa pelo trigram
	getJSON(t, srv.URL+"/v1/acts?q=345.678/0001", &res)
	if res.Total != 1 || res.Items[0].Type != "contrato" || !strings.Contains(res.Items[0].Snippet, "⟦345.678/0001⟧") {
		t.Fatalf("busca por substring inesperada: %+v", res)
	}

	// 2d. Linha do tempo por CNPJ (formatado ou só dígitos) e estatísticas por mês
	var company struct {
		CNPJ            string         `json:"cnpj"`
		TotalValueCents int64          `json:"total_value_cents"`
		CountByType     map[string]int `json:"count_by_type"`
		Acts            []struct{ Type, Snippet string }
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/12.345.678%2F0001-90", &company)
	if company.CNPJ != "12345678000190" || company.TotalValueCents != 12000000 || company.CountByType["contrato"] != 1 ||
		len(company.Acts) != 1 || !strings.Contains(company.Acts[0].Snippet, "⟦12.345.678/0001-90⟧") {
		t.Fatalf("relatório por CNPJ inesperado: %+v", company)
	}
	getJSON(t, srv.URL+"/v1/entities/cnpj/12345678000190", &company)
	if len(company.Acts) != 1 {
		t.Fatalf("CNPJ só com dígitos deveria dar o mesmo resultado: %+v", company)
	}
	if r, err := http.Get(srv.URL + "/v1/entities/cnpj/123"); err != nil || r.StatusCode != http.StatusBadRequest {
		t.Fatalf("CNPJ inválido deve dar 400: %v %v", r, err)
	}
	var stats struct {
		Group string
		Items []struct {
			Month string
			Count int
		}
	}
	getJSON(t, srv.URL+"/v1/stats/acts?group=month", &stats)
	if stats.Group != "month" || len(stats.Items) != 1 || stats.Items[0].Month != "2026-09" || stats.Items[0].Count != 5 {
		t.Fatalf("estatísticas inesperadas: %+v", stats)
	}
	getJSON(t, srv.URL+"/v1/stats/acts?type=contrato&from=2026-09-01&to=2026-09-30", &stats)
	if len(stats.Items) != 1 || stats.Items[0].Count != 1 {
		t.Fatalf("estatísticas filtradas inesperadas: %+v", stats)
	}
	getJSON(t, srv.URL+"/v1/stats/acts?q=medicamento", &stats)
	if len(stats.Items) != 1 || stats.Items[0].Count != 1 {
		t.Fatalf("estatísticas devem respeitar o termo de busca: %+v", stats)
	}

	// 2e. Termo curto sem dígito não casa por substring (evita inundar alertas)
	getJSON(t, srv.URL+"/v1/acts?q=sil", &res)
	if res.Total != 0 {
		t.Fatalf("'sil' não deveria casar 'SILVA' por substring: %+v", res)
	}
	if hits, err := acts.SearchInGazette(ctx, pub.ids[0], ""); err != nil || len(hits) != 0 {
		t.Fatalf("query vazia não pode casar atos: %v %v", hits, err)
	}

	// 3. Inscrição -> confirmação -> alerta (sem duplicar)
	post(t, srv.URL+"/v1/subscriptions", `{"email":"rep@jornal.com","query":"medicamentos"}`, http.StatusAccepted)
	token := strings.Split(strings.Split(box.msgs[0].HTML, "token=")[1], `"`)[0]
	post(t, srv.URL+"/v1/subscriptions/confirm", `{"token":"`+token+`"}`, http.StatusOK)

	match := usecase.NewMatchSubscriptions(gaz, acts, subs, nlog, notifier)
	for i := 0; i < 2; i++ {
		if err := match.Execute(ctx, pub.ids[0]); err != nil {
			t.Fatal(err)
		}
	}
	if len(box.msgs) != 2 {
		t.Fatalf("esperava 1 confirmação + 1 alerta, veio %d e-mails", len(box.msgs))
	}
}

// openTestDB cria o banco de teste quando ele ainda não existe (localmente
// usamos um banco separado para não misturar com as edições ingeridas).
func openTestDB(t *testing.T, ctx context.Context, url string) *sql.DB {
	t.Helper()
	db, err := postgres.Open(ctx, url)
	if err == nil {
		return db
	}
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != "3D000" {
		t.Fatal(err)
	}
	u, perr := neturl.Parse(url)
	if perr != nil {
		t.Fatal(perr)
	}
	name := strings.TrimPrefix(u.Path, "/")
	u.Path = "/postgres"
	admin, err := postgres.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.ExecContext(ctx, `CREATE DATABASE `+pq.QuoteIdentifier(name)); err != nil {
		t.Fatal(err)
	}
	db, err = postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func getJSON(t *testing.T, url string, v any) {
	t.Helper()
	r, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
}

func post(t *testing.T, url, body string, want int) {
	t.Helper()
	r, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != want {
		t.Fatalf("POST %s: esperava %d, veio %d", url, want, r.StatusCode)
	}
}
