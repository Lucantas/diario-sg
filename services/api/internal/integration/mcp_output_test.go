//go:build integration

package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

func manyValuesGazette() string {
	var b strings.Builder
	b.WriteString("EXTRATO DO CONTRATO Nº 1/2026\nObjeto: merenda escolar. Valor global: R$ 50.000,00.\n")
	for i := 1; i <= 25; i++ {
		fmt.Fprintf(&b, "Item %d: R$ %d,00.\n", i, i)
	}
	return b.String()
}

type compactAct struct {
	GazetteID   string  `json:"edicao_id"`
	Position    int     `json:"posicao"`
	ValuesCents []int64 `json:"valores_centavos"`
	ValuesTotal int     `json:"valores_total"`
	InRange     []int64 `json:"valores_na_faixa_centavos"`
}

type compactSearch struct {
	Total    int          `json:"total"`
	Acts     []compactAct `json:"atos"`
	Coverage []any        `json:"cobertura"`
	Alerts   []string     `json:"alertas_coleta"`
}

func TestMCPSearchSaysWhichValuesAreInRangeAndTruncatesTheList(t *testing.T) {
	srv, _ := newServerFor(t, manyValuesGazette())
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	out, _ := call[compactSearch](t, session, "buscar_atos", map[string]any{"valor_min": 40000, "valor_max": 62000})

	if out.Total != 1 || len(out.Acts) != 1 {
		t.Fatalf("esperava o extrato: %+v", out)
	}
	act := out.Acts[0]
	if len(act.InRange) != 1 || act.InRange[0] != 5000000 {
		t.Errorf("valor que casou com a faixa: %v", act.InRange)
	}
	if len(act.ValuesCents) != 20 || act.ValuesTotal != 26 {
		t.Errorf("lista de valores deveria vir truncada em 20 de 26: %d de %d", len(act.ValuesCents), act.ValuesTotal)
	}
	if len(out.Coverage) == 0 {
		t.Error("a primeira página traz a cobertura")
	}

	next, _ := call[compactSearch](t, session, "buscar_atos", map[string]any{"deslocamento": 1})
	if next.Coverage != nil {
		t.Errorf("páginas seguintes não repetem a cobertura: %+v", next.Coverage)
	}

	read, _ := call[struct {
		Coverage []any  `json:"cobertura"`
		Text     string `json:"texto"`
	}](t, session, "ler_ato", map[string]any{"edicao_id": act.GazetteID, "posicao": act.Position})
	if read.Text == "" || read.Coverage != nil {
		t.Errorf("ler_ato traz o texto e não repete a cobertura: %+v", read.Coverage)
	}
}

func TestMCPWarnsAboutAFailedCollectionInEveryAnswer(t *testing.T) {
	srv, db := newServerFor(t, manyValuesGazette())
	start := time.Date(2026, 9, 23, 21, 30, 0, 0, time.UTC)
	failed := domain.FetchRun{ID: "44444444-4444-4444-8444-444444444444", Source: domain.SourceDiarioCamara,
		RequestedFrom: start, RequestedTo: start, Failed: 1, Error: "context canceled", StartedAt: start, FinishedAt: start.Add(time.Minute)}
	if err := usecase.NewRecordFetchRun(postgres.NewFetchRunRepo(db)).Execute(context.Background(), failed); err != nil {
		t.Fatal(err)
	}
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	out, _ := call[compactSearch](t, session, "buscar_atos", map[string]any{"deslocamento": 5})

	if len(out.Alerts) != 1 || !strings.Contains(out.Alerts[0], "Câmara") || !strings.Contains(out.Alerts[0], "context canceled") {
		t.Fatalf("alerta da coleta que falhou: %v", out.Alerts)
	}
}

const publicBodyGazette = "EXTRATO DO CONTRATO Nº 7/2026\n" +
	"Partes: MUNICÍPIO DE SÃO GONÇALO, CNPJ 28.636.579/0001-00, e ALL FOOD SERVIÇOS LTDA, CNPJ 01.742.126/0001-02.\n" +
	"Valor global: R$ 50.000,00."

func TestMCPMarksPublicBodyCNPJs(t *testing.T) {
	srv, _ := newServerFor(t, publicBodyGazette)
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	found, _ := call[struct {
		Acts []struct {
			CNPJs       []string `json:"cnpjs"`
			PublicCNPJs []string `json:"cnpjs_orgaos_publicos"`
		} `json:"atos"`
	}](t, session, "buscar_atos", map[string]any{})
	entity, _ := call[struct {
		PublicBody string `json:"orgao_publico"`
	}](t, session, "entidade", map[string]any{"numero": "28.636.579/0001-00"})
	supplier, _ := call[struct {
		PublicBody string `json:"orgao_publico"`
	}](t, session, "entidade", map[string]any{"numero": "01.742.126/0001-02"})

	if len(found.Acts) != 1 || len(found.Acts[0].CNPJs) != 2 || len(found.Acts[0].PublicCNPJs) != 1 {
		t.Fatalf("o ato deveria separar o CNPJ do Município: %+v", found.Acts)
	}
	if entity.PublicBody != "Município de São Gonçalo" || supplier.PublicBody != "" {
		t.Fatalf("orgao_publico: Município %q, fornecedor %q", entity.PublicBody, supplier.PublicBody)
	}
}
