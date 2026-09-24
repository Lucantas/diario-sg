//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type singlePage struct{}

func (singlePage) ExtractPage(_ context.Context, r io.Reader, _ string, page int) (string, int, error) {
	if page != 1 {
		return "", 1, fmt.Errorf("página %d de 1: %w", page, domain.ErrInvalidInput)
	}
	b, err := io.ReadAll(r)
	return string(b), 1, err
}

func TestMCPShowsTheRawPageOfTheArchivedPDF(t *testing.T) {
	srv, _ := newServerFor(t, contractingGazette)
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	found, _ := call[struct {
		Acts []struct {
			GazetteID string `json:"edicao_id"`
		} `json:"atos"`
	}](t, session, "buscar_atos", map[string]any{})

	page, _ := call[struct {
		Page   int    `json:"pagina"`
		Pages  int    `json:"paginas_total"`
		Text   string `json:"texto"`
		Source struct {
			SHA256 string `json:"sha256"`
			URL    string `json:"url"`
		} `json:"fonte"`
	}](t, session, "pagina_original", map[string]any{"edicao_id": found.Acts[0].GazetteID, "pagina": 1})
	_, beyond := call[struct{}](t, session, "pagina_original", map[string]any{"edicao_id": found.Acts[0].GazetteID, "pagina": 2})

	if page.Page != 1 || page.Pages != 1 || !strings.Contains(page.Text, "LM CURSOS") || page.Source.SHA256 == "" || !strings.HasSuffix(page.Source.URL, "#page=1") {
		t.Fatalf("página crua: %+v", page)
	}
	if !beyond.IsError {
		t.Fatal("página além da última é erro de entrada")
	}
}
