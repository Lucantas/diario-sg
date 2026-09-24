//go:build integration

package integration

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func fetch(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	r, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return r, string(body)
}

func TestExportCSV(t *testing.T) {
	srv := newInvestigatorServer(t)

	r, body := fetch(t, srv.URL+"/v1/acts/export")

	if r.StatusCode != http.StatusOK || !strings.HasPrefix(body, "\xef\xbb\xbf") ||
		r.Header.Get("X-Total-Count") != "3" || r.Header.Get("X-Export-Truncated") != "false" ||
		!strings.HasPrefix(r.Header.Get("Content-Type"), "text/csv") ||
		!strings.Contains(r.Header.Get("Content-Disposition"), `attachment; filename="diario-sg-busca-`) {
		t.Fatalf("resposta inesperada: %d %v", r.StatusCode, r.Header)
	}
	records := readCSV(t, body)
	if len(records) != 4 || strings.Join(records[0], ";") != "data;edicao;extra;tipo;orgao;orgao_nome;titulo;pagina_inicio;pagina_fim;valores_reais;cnpjs;pdf_original;pdf_arquivado;sha256_pdf;avisos;texto;fonte" {
		t.Fatalf("CSV inesperado: %q", records)
	}
	var third []string
	for _, rec := range records[1:] {
		if rec[6] == "EXTRATO DO CONTRATO Nº 3/2026" {
			third = rec
		}
	}
	if third == nil || third[9] != "200000,00 | 20000,00" || third[7] != "1" ||
		!strings.HasPrefix(third[12], "https://web.exemplo/api/v1/gazettes/") || !strings.HasSuffix(third[12], "/pdf#page=1") ||
		third[11] != "https://exemplo/9.pdf#page=1" || !strings.Contains(third[15], "mensal R$ 20.000,00") {
		t.Fatalf("linha do contrato 3 inesperada: %q", third)
	}

	_, body = fetch(t, srv.URL+"/v1/acts/export?format=csv&min_value=40000")
	if got := readCSV(t, body); len(got) != 3 {
		t.Fatalf("filtro de valor deveria exportar 2 atos além do cabeçalho, veio %d registros", len(got))
	}
}

func TestExportJSON(t *testing.T) {
	srv := newInvestigatorServer(t)

	r, body := fetch(t, srv.URL+"/v1/acts/export?format=json&q=limpeza")

	var out struct {
		Total     int  `json:"total"`
		Truncated bool `json:"truncated"`
		Items     []struct {
			Title          string  `json:"title"`
			Body           string  `json:"body"`
			Snippet        *string `json:"snippet"`
			ArchivedPDFURL string  `json:"archived_pdf_url"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, body)
	}
	if r.StatusCode != http.StatusOK || out.Total != 2 || out.Truncated || len(out.Items) != 2 {
		t.Fatalf("exportação inesperada: %d %+v", r.StatusCode, out)
	}
	for _, it := range out.Items {
		if it.Snippet != nil || it.Body == "" || !strings.HasSuffix(it.ArchivedPDFURL, "/pdf#page=1") {
			t.Fatalf("item inesperado: %+v", it)
		}
	}

	r, _ = fetch(t, srv.URL+"/v1/acts/export?format=json&q=inexistentexyz")
	if r.StatusCode != http.StatusOK {
		t.Fatalf("exportação vazia deve dar 200, veio %d", r.StatusCode)
	}

	r, _ = fetch(t, srv.URL+"/v1/acts/export?format=xml")
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("formato desconhecido deve dar 400, veio %d", r.StatusCode)
	}
}

func TestExportJSONDoesNotExposePhaseOrMentions(t *testing.T) {
	srv, _ := newEntityServer(t)

	_, body := fetch(t, srv.URL+"/v1/acts/export?format=json&q="+url.QueryEscape("30/SEMAD/2023"))

	if !strings.Contains(body, "SEMAD") {
		t.Fatalf("esperava achar o contrato exportado: %s", body)
	}
	if strings.Contains(body, `"phase"`) || strings.Contains(body, `"mentions"`) {
		t.Fatalf("exportação JSON não deve trazer phase nem mentions: %s", body)
	}
}

func readCSV(t *testing.T, body string) [][]string {
	t.Helper()
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(body, "\xef\xbb\xbf")))
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return records
}
