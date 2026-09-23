//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type bearer struct{ key string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.key)
	return http.DefaultTransport.RoundTrip(r)
}

func issueKey(t *testing.T, url, body string) (int, string) {
	t.Helper()
	r, err := http.Post(url+"/v1/mcp/keys", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var out struct {
		Key    string `json:"key"`
		MCPURL string `json:"mcp_url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&out)
	if r.StatusCode == http.StatusCreated && out.MCPURL != "https://web.exemplo/api/mcp" {
		t.Fatalf("mcp_url inesperada: %q", out.MCPURL)
	}
	return r.StatusCode, out.Key
}

func connect(t *testing.T, url, key string) (*sdk.ClientSession, error) {
	t.Helper()
	client := sdk.NewClient(&sdk.Implementation{Name: "teste", Version: "0"}, nil)
	return client.Connect(context.Background(), &sdk.StreamableClientTransport{
		Endpoint: url + "/mcp", HTTPClient: &http.Client{Transport: bearer{key}}, MaxRetries: -1, DisableStandaloneSSE: true,
	}, nil)
}

func call[T any](t *testing.T, s *sdk.ClientSession, tool string, args map[string]any) (T, *sdk.CallToolResult) {
	t.Helper()
	var out T
	res, err := s.CallTool(context.Background(), &sdk.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	if res.IsError {
		return out, res
	}
	raw, _ := json.Marshal(res.StructuredContent)
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	return out, res
}

const toolsList = `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

func statusOf(t *testing.T, method, url, key string) int {
	t.Helper()
	return statusWithBody(t, method, url, key, toolsList)
}

func statusWithBody(t *testing.T, method, url, key, body string) int {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	return r.StatusCode
}

type actRef struct {
	GazetteID string `json:"edicao_id"`
	Position  int    `json:"posicao"`
	Title     string `json:"titulo"`
	Sources   []struct {
		ArchivedCopy string `json:"copia_arquivada"`
	} `json:"fontes"`
}

func TestMCPWithPerUserKey(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)

	if got := statusOf(t, http.MethodPost, srv.URL+"/mcp", ""); got != http.StatusUnauthorized {
		t.Fatalf("sem chave deveria ser 401, veio %d", got)
	}
	if got := statusOf(t, http.MethodPost, srv.URL+"/mcp", "dsg_inventadainventadainventada"); got != http.StatusUnauthorized {
		t.Fatalf("chave inventada deveria ser 401, veio %d", got)
	}

	code, key := issueKey(t, srv.URL, "")
	if code != http.StatusCreated || !strings.HasPrefix(key, "dsg_") {
		t.Fatalf("emissão: %d %q", code, key)
	}
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	batch := " [" + toolsList + "," + toolsList + "]"
	if got := statusWithBody(t, http.MethodPost, srv.URL+"/mcp", key, batch); got != http.StatusBadRequest {
		t.Fatalf("lote JSON-RPC deveria ser 400, veio %d", got)
	}

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("%s deveria ser só leitura", tool.Name)
		}
	}
	sort.Strings(names)
	if strings.Join(names, ",") != "buscar_atos,entidade,fontes,ler_ato" {
		t.Fatalf("ferramentas: %v", names)
	}

	found, _ := call[struct {
		Total    int      `json:"total"`
		Acts     []actRef `json:"atos"`
		Coverage struct {
			From string `json:"de"`
		} `json:"cobertura"`
	}](t, session, "buscar_atos", map[string]any{"consulta": "medicamentos", "limite": 50})
	if found.Total != 1 || len(found.Acts) != 1 || found.Coverage.From != "2026-09-18" ||
		!strings.HasPrefix(found.Acts[0].Sources[0].ArchivedCopy, "https://web.exemplo/api/v1/gazettes/") {
		t.Fatalf("busca inesperada: %+v", found)
	}

	act, _ := call[struct {
		Text     string `json:"texto"`
		Citation string `json:"citacao"`
	}](t, session, "ler_ato", map[string]any{"edicao_id": found.Acts[0].GazetteID, "posicao": found.Acts[0].Position})
	if !strings.Contains(act.Text, "12.345.678/0001-90") || !strings.Contains(act.Citation, "ed. 9, 18 set. 2026") {
		t.Fatalf("ato inesperado: %+v", act)
	}
	if _, res := call[struct{}](t, session, "ler_ato", map[string]any{"edicao_id": found.Acts[0].GazetteID, "posicao": 99}); !res.IsError {
		t.Fatal("posição inexistente deveria voltar como erro da ferramenta")
	}

	company, _ := call[struct {
		TotalActs  int   `json:"total_atos"`
		TotalCents int64 `json:"soma_valores_centavos"`
	}](t, session, "entidade", map[string]any{"cnpj": "12345678000190"})
	if company.TotalActs != 1 || company.TotalCents != 12000000 {
		t.Fatalf("entidade inesperada: %+v", company)
	}
	if _, res := call[struct{}](t, session, "entidade", map[string]any{"cnpj": "123"}); !res.IsError {
		t.Fatal("CNPJ inválido deveria voltar como erro da ferramenta")
	}

	sources, _ := call[struct {
		Sources []struct {
			Gazettes int `json:"edicoes"`
			Acts     int `json:"atos"`
		} `json:"fontes"`
	}](t, session, "fontes", nil)
	if len(sources.Sources) != 1 || sources.Sources[0].Gazettes != 1 || sources.Sources[0].Acts == 0 {
		t.Fatalf("fontes inesperadas: %+v", sources)
	}

	rows, err := db.Query(`SELECT tool, calls FROM api_key_usage ORDER BY tool`)
	if err != nil {
		t.Fatal(err)
	}
	usage := map[string]int{}
	for rows.Next() {
		var tool string
		var calls int
		if err := rows.Scan(&tool, &calls); err != nil {
			t.Fatal(err)
		}
		usage[tool] = calls
	}
	rows.Close()
	if usage["buscar_atos"] != 1 || usage["ler_ato"] != 2 || usage["entidade"] != 2 || usage["fontes"] != 1 {
		t.Fatalf("uso registrado: %v", usage)
	}

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/v1/mcp/keys", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusNoContent {
		t.Fatalf("revogar deveria ser 204, veio %d", r.StatusCode)
	}
	if got := statusOf(t, http.MethodPost, srv.URL+"/mcp", key); got != http.StatusUnauthorized {
		t.Fatalf("chave revogada deveria ser 401, veio %d", got)
	}

	var stored int
	if code, _ := issueKey(t, srv.URL, `{"website":"http://spam"}`); code != http.StatusCreated {
		t.Fatalf("campo isca deveria parecer sucesso, veio %d", code)
	}
	if err := db.QueryRow(`SELECT count(*) FROM api_keys`).Scan(&stored); err != nil || stored != 1 {
		t.Fatalf("campo isca não grava chave: %d %v", stored, err)
	}
	if code, _ := issueKey(t, srv.URL, ""); code != http.StatusCreated {
		t.Fatalf("terceira chave da hora deveria passar, veio %d", code)
	}
	if code, _ := issueKey(t, srv.URL, ""); code != http.StatusTooManyRequests {
		t.Fatalf("quarta chave da hora deveria ser 429, veio %d", code)
	}
}
