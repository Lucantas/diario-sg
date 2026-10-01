package pdf

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"maps"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type memOCRCache struct {
	pages    map[string]map[int]string
	saved    map[string]map[int]string
	failLoad bool
	failSave bool
}

func (c *memOCRCache) Load(_ context.Context, key string) (map[int]string, error) {
	if c.failLoad {
		return nil, errors.New("bucket fora do ar")
	}
	return maps.Clone(c.pages[key]), nil
}

func (c *memOCRCache) Save(_ context.Context, key string, pages map[int]string) error {
	if c.failSave {
		return errors.New("sem permissão")
	}
	if c.saved == nil {
		c.saved = map[string]map[int]string{}
	}
	c.saved[key] = maps.Clone(pages)
	return nil
}

type countingOCR struct{ pages []int }

func (o *countingOCR) read(_ context.Context, _, _ string, page int) (string, error) {
	o.pages = append(o.pages, page)
	return "LIDO PELO OCR", nil
}

func withCache(cache OCRCache, ocr *countingOCR) PDFToText {
	p := New().WithCache(cache, nil)
	p.ocr = ocr.read
	return p
}

func ocrKey(pdf []byte) string {
	sum := sha256.Sum256(pdf)
	return ocrSettingsVersion + "/" + hex.EncodeToString(sum[:])
}

func TestCachedOCRPagesAreNotReadAgain(t *testing.T) {
	requireOCRCandidates(t)
	pdf := scannedPDF(t, 2)
	cache := &memOCRCache{pages: map[string]map[int]string{ocrKey(pdf): {1: "PAGINA UM DO CACHE", 2: "PAGINA DOIS DO CACHE"}}}
	ocr := &countingOCR{}

	got, err := withCache(cache, ocr).Extract(context.Background(), bytes.NewReader(pdf), domain.SourceDiarioPrefeitura)

	if err != nil {
		t.Fatal(err)
	}
	if len(ocr.pages) != 0 || cache.saved != nil {
		t.Errorf("não devia rodar OCR nem regravar: ocr %v, gravado %v", ocr.pages, cache.saved)
	}
	if !strings.Contains(got.Text, "PAGINA UM DO CACHE") || !strings.Contains(got.Text, "PAGINA DOIS DO CACHE") || !slices.Equal(got.OCRPages, []int{1, 2}) {
		t.Errorf("texto do cache: %q %v", got.Text, got.OCRPages)
	}
}

func TestOnlyMissingPagesAreReadAndTheCacheIsCompleted(t *testing.T) {
	requireOCRCandidates(t)
	pdf := scannedPDF(t, 2)
	cache := &memOCRCache{pages: map[string]map[int]string{ocrKey(pdf): {1: "PAGINA UM DO CACHE"}}}
	ocr := &countingOCR{}

	got, err := withCache(cache, ocr).Extract(context.Background(), bytes.NewReader(pdf), domain.SourceDiarioPrefeitura)

	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ocr.pages, []int{2}) {
		t.Errorf("OCR só da página 2: %v", ocr.pages)
	}
	if want := map[int]string{1: "PAGINA UM DO CACHE", 2: "LIDO PELO OCR"}; !maps.Equal(cache.saved[ocrKey(pdf)], want) {
		t.Errorf("cache gravado: %v", cache.saved)
	}
	if !slices.Equal(got.OCRPages, []int{1, 2}) {
		t.Errorf("páginas: %v", got.OCRPages)
	}
}

func TestAnotherPDFDoesNotUseTheCache(t *testing.T) {
	requireOCRCandidates(t)
	pdf := scannedPDF(t, 1)
	cache := &memOCRCache{pages: map[string]map[int]string{ocrKey(append([]byte{}, pdf[:len(pdf)-1]...)): {1: "OUTRO PDF"}}}
	ocr := &countingOCR{}

	got, err := withCache(cache, ocr).Extract(context.Background(), bytes.NewReader(pdf), domain.SourceDiarioPrefeitura)

	if err != nil || strings.Contains(got.Text, "OUTRO PDF") || !slices.Equal(ocr.pages, []int{1}) {
		t.Errorf("PDF diferente: %q %v %v", got.Text, ocr.pages, err)
	}
	if _, ok := cache.saved[ocrKey(pdf)]; !ok {
		t.Errorf("devia gravar com a chave do PDF lido: %v", cache.saved)
	}
}

func TestCacheFailuresFallBackToOCR(t *testing.T) {
	requireOCRCandidates(t)
	pdf := scannedPDF(t, 1)
	for name, cache := range map[string]*memOCRCache{
		"leitura":  {failLoad: true},
		"gravação": {failSave: true},
	} {
		ocr := &countingOCR{}

		got, err := withCache(cache, ocr).Extract(context.Background(), bytes.NewReader(pdf), domain.SourceDiarioPrefeitura)

		if err != nil || !strings.Contains(got.Text, "LIDO PELO OCR") || !slices.Equal(ocr.pages, []int{1}) {
			t.Errorf("falha na %s do cache: %q %v %v", name, got.Text, ocr.pages, err)
		}
	}
}

func TestExtractPageUsesTheCacheOfTheWholeEdition(t *testing.T) {
	requireOCRCandidates(t)
	pdf := scannedPDF(t, 2)
	cache := &memOCRCache{pages: map[string]map[int]string{ocrKey(pdf): {2: "PAGINA DOIS DO CACHE"}}}
	ocr := &countingOCR{}

	got, _, err := withCache(cache, ocr).ExtractPage(context.Background(), bytes.NewReader(pdf), domain.SourceDiarioPrefeitura, 2)

	if err != nil || len(ocr.pages) != 0 || !strings.Contains(got.Text, "PAGINA DOIS DO CACHE") {
		t.Errorf("página 2 pelo cache: %q %v %v", got.Text, ocr.pages, err)
	}
}

func requireOCRCandidates(t *testing.T) {
	t.Helper()
	requirePoppler(t)
	for _, bin := range []string{"pdftoppm", "pdfimages"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s não instalado", bin)
		}
	}
}
