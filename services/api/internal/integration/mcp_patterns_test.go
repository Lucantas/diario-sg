//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpPatterns struct {
	Patterns []struct {
		ID       string `json:"id"`
		Rule     string `json:"regra"`
		Findings int    `json:"achados_total"`
	} `json:"padroes"`
	Findings []struct {
		Pattern  string `json:"padrao"`
		Title    string `json:"titulo"`
		Entities []struct {
			Kind string `json:"tipo"`
			Key  string `json:"chave"`
		} `json:"entidades"`
		Acts []struct {
			GazetteID string `json:"edicao_id"`
			URL       string `json:"url"`
			Archived  string `json:"copia_arquivada"`
		} `json:"atos"`
		ActsTotal int `json:"atos_total"`
	} `json:"achados"`
	Total  int    `json:"achados_encontrados"`
	Entity string `json:"entidade"`
}

func TestMCPPatternsFilterFindingsByEntity(t *testing.T) {
	srv, db := newServerFor(t, semmaDispensa2020)
	setOnlyGazetteDate(t, db, "2020-02-19")
	indexAt(t, db, time.Date(2020, 5, 22, 0, 0, 0, 0, time.UTC), semadDispensa2020)
	indexAt(t, db, time.Date(2024, 10, 3, 0, 0, 0, 0, time.UTC), semtranRatificacao2024)
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	catalog, _ := call[mcpPatterns](t, session, "padroes", nil)
	if len(catalog.Patterns) != 15 || catalog.Patterns[0].ID != "fracionamento_dispensa" || catalog.Patterns[0].Findings != 1 || catalog.Patterns[0].Rule == "" ||
		catalog.Total != 1 || len(catalog.Findings[0].Acts) != 0 {
		t.Fatalf("catálogo: %+v", catalog.Patterns)
	}

	byCNPJ, _ := call[mcpPatterns](t, session, "padroes", map[string]any{"cnpj": "53.775.862/0001-52"})
	if len(byCNPJ.Findings) != 1 || byCNPJ.Total != 1 || len(byCNPJ.Patterns) != 1 || byCNPJ.Entity != "cnpj 53775862000152" {
		t.Fatalf("por CNPJ: %+v", byCNPJ)
	}
	f := byCNPJ.Findings[0]
	if f.Pattern != "fracionamento_dispensa" || f.ActsTotal != 2 || len(f.Acts) != 2 || !strings.HasPrefix(f.Acts[0].Archived, "https://web.exemplo/") ||
		f.Acts[0].URL == "" || len(f.Entities) != 3 {
		t.Fatalf("achado: %+v", f)
	}
	processo := "215772019"
	hasProcess := false
	for _, e := range f.Entities {
		hasProcess = hasProcess || (e.Kind == "processo" && e.Key == processo)
	}
	if !hasProcess {
		t.Fatalf("achado sem o processo %s: %+v", processo, f.Entities)
	}

	byProcess, _ := call[mcpPatterns](t, session, "padroes", map[string]any{"processo": "21.577/2019"})
	if len(byProcess.Findings) != 1 || byProcess.Entity != "processo "+processo {
		t.Errorf("pelo processo escrito como no Diário: %+v", byProcess)
	}
	entity, _ := call[struct {
		Key       string `json:"chave"`
		TotalActs int    `json:"total_atos"`
	}](t, session, "entidade", map[string]any{"tipo": "processo", "numero": processo})
	if entity.Key != processo || entity.TotalActs == 0 {
		t.Errorf("entidade com a chave do achado: %+v", entity)
	}
	if _, res := call[mcpPatterns](t, session, "padroes", map[string]any{"pular": 3}); res == nil || !res.IsError {
		t.Error("pular sem padrão nem entidade aceito")
	}
	none, _ := call[mcpPatterns](t, session, "padroes", map[string]any{"cnpj": "18.657.198/0001-46"})
	if len(none.Findings) != 0 || len(none.Patterns) != 0 {
		t.Errorf("CNPJ sem padrão: %+v", none)
	}
	if _, res := call[mcpPatterns](t, session, "padroes", map[string]any{"padrao": "inventado"}); res == nil || !res.IsError {
		t.Error("padrão inexistente aceito")
	}
	if _, res := call[mcpPatterns](t, session, "padroes", map[string]any{"cnpj": "53775862000152", "contrato": "55/2026"}); res == nil || !res.IsError {
		t.Error("duas entidades aceitas")
	}
}

func TestMCPPromptsGuideTheInvestigation(t *testing.T) {
	srv, _ := newServerFor(t, gazetteText)
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	ctx := context.Background()

	list, err := session.ListPrompts(ctx, nil)
	if err != nil || len(list.Prompts) != 2 {
		t.Fatalf("prompts: %+v %v", list, err)
	}
	supplier, err := session.GetPrompt(ctx, &sdk.GetPromptParams{Name: "investigar_fornecedor", Arguments: map[string]string{"cnpj": "53.775.862/0001-52"}})
	if err != nil {
		t.Fatal(err)
	}
	text := supplier.Messages[0].Content.(*sdk.TextContent).Text
	if !strings.Contains(text, "entidade com tipo cnpj e numero 53775862000152") || !strings.Contains(text, "padroes com cnpj 53775862000152") {
		t.Errorf("roteiro do fornecedor: %s", text)
	}
	contract, err := session.GetPrompt(ctx, &sdk.GetPromptParams{Name: "seguir_contrato", Arguments: map[string]string{"numero": "8421/2023"}})
	if err != nil {
		t.Fatal(err)
	}
	text = contract.Messages[0].Content.(*sdk.TextContent).Text
	if !strings.Contains(text, "tipo processo e numero 8421/2023") || !strings.Contains(text, "tipo contrato e numero 8421/2023") {
		t.Errorf("roteiro do contrato: %s", text)
	}
	if _, err := session.GetPrompt(ctx, &sdk.GetPromptParams{Name: "investigar_fornecedor", Arguments: map[string]string{"cnpj": "123"}}); err == nil {
		t.Error("CNPJ inválido aceito")
	}
	if !strings.Contains(text, "pagamentos e contratacoes com processo") {
		t.Errorf("roteiro do contrato sem o portal e o mural: %s", text)
	}
}

func TestMCPResourceDescribesTheSources(t *testing.T) {
	srv, _ := newServerFor(t, gazetteText)
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	ctx := context.Background()

	list, err := session.ListResources(ctx, nil)
	if err != nil || len(list.Resources) != 1 || list.Resources[0].URI != "diario-sg://fontes" {
		t.Fatalf("recursos: %+v %v", list, err)
	}
	read, err := session.ReadResource(ctx, &sdk.ReadResourceParams{URI: "diario-sg://fontes"})
	if err != nil {
		t.Fatal(err)
	}
	if text := read.Contents[0].Text; !strings.Contains(text, "## O que não está na base") || !strings.Contains(text, "`pagamentos`") {
		t.Errorf("descrição das fontes: %s", text)
	}
}
