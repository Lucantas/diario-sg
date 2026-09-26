//go:build integration

package integration

import (
	"context"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

type scannedExtractor struct{}

func (scannedExtractor) Extract(_ context.Context, r io.Reader, _ string) (domain.ExtractedText, error) {
	b, err := io.ReadAll(r)
	return domain.ExtractedText{Text: string(b), OCRPages: []int{1}}, err
}

func TestActsReadByOCRCarryTheWarning(t *testing.T) {
	srv, db := newServerFor(t, gazetteText)
	day := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	reindex := usecase.NewReindexGazettes(postgres.NewGazetteRepo(db), stringStore(gazetteText), scannedExtractor{}, parser.Set{}, entities.New())

	if _, err := reindex.Execute(context.Background(), day, day); err != nil {
		t.Fatal(err)
	}

	var res struct {
		Items []struct {
			Warnings []string `json:"warnings"`
		} `json:"items"`
	}
	getJSON(t, srv.URL+"/v1/acts?q=medicamento", &res)
	if len(res.Items) != 1 || !slices.Contains(res.Items[0].Warnings, domain.WarningReadByOCR) {
		t.Errorf("aviso de OCR na busca: %+v", res)
	}
	var flagged int
	if err := db.QueryRow(`SELECT count(*) FROM acts WHERE read_by_ocr`).Scan(&flagged); err != nil || flagged != 5 {
		t.Errorf("atos marcados: %d %v", flagged, err)
	}
}
