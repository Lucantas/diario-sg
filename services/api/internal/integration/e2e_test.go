//go:build integration

// Teste de integração: Postgres real + camada HTTP + casos de uso.
// Rode com: TEST_DATABASE_URL=postgres://... go test -tags integration ./internal/integration/
// No CI, o GitHub Actions sobe um Postgres como service container.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/email"
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
Contratada: Empresa Exemplo LTDA. Objeto: fornecimento de medicamentos.`

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
	db, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := postgres.Migrate(ctx, db, migrations.FS); err != nil {
		t.Fatal(err)
	}

	gaz, acts := postgres.NewGazetteRepo(db), postgres.NewActRepo(db)
	subs, nlog := postgres.NewSubscriptionRepo(db), postgres.NewNotificationLog(db)
	pub := &recPub{}
	box := &inbox{}
	notifier := email.NewNotifier(box, "https://web.exemplo")

	// 1. Indexação (duas vezes: tem que ser idempotente)
	idx := usecase.NewIndexGazette(gaz, textStore{}, passthroughExtractor{}, parser.New(), pub)
	in := usecase.IndexGazetteInput{EditionNumber: "1", PublishedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
		SourceURL: "https://exemplo/1.pdf", StoragePath: "1.pdf", Checksum: strings.Repeat("b", 64)}
	for i := 0; i < 2; i++ {
		if err := idx.Execute(ctx, in); err != nil {
			t.Fatal(err)
		}
	}

	api := &httpapi.API{Search: usecase.NewSearchActs(acts), Gazette: usecase.NewGetGazette(gaz, acts),
		Subscriptions: usecase.NewSubscriptions(subs, notifier), Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	srv := httptest.NewServer(api.Routes())
	defer srv.Close()

	// 2. Busca textual em português (stemming: "medicamento" acha "medicamentos")
	var res struct {
		Items []struct{ Type, Snippet string } `json:"items"`
		Total int                              `json:"total"`
	}
	getJSON(t, srv.URL+"/v1/acts?q=medicamento", &res)
	if res.Total != 1 || res.Items[0].Type != "contrato" || !strings.Contains(res.Items[0].Snippet, "⟦") {
		t.Fatalf("busca inesperada: %+v", res)
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
