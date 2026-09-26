package pdf

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const scannedLine = "DECRETO LEGISLATIVO 06 2020"

func requireOCR(t *testing.T) {
	t.Helper()
	requirePoppler(t)
	for _, bin := range []string{"pdftoppm", "pdfimages", "tesseract"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s não instalado", bin)
		}
	}
	langs, err := exec.Command("tesseract", "--list-langs").CombinedOutput()
	if err != nil || !strings.Contains(string(langs), "\npor") {
		t.Skip("tesseract sem o português")
	}
}

func textPDF(line string) []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 300] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		stream(fmt.Sprintf("BT /F1 30 Tf 30 150 Td (%s) Tj ET", line)),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	return pdfOf(objects)
}

func pdfOf(objects []string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for i, o := range objects {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return b.Bytes()
}

func grayPixels(t *testing.T, pdf []byte) (int, int, []byte) {
	t.Helper()
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(in, pdf, 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("pdftoppm", "-r", "150", "-gray", "-singlefile", in, filepath.Join(dir, "out")).CombinedOutput(); err != nil {
		t.Fatalf("pdftoppm: %v %s", err, out)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "out.pgm"))
	if err != nil {
		t.Fatal(err)
	}
	r := bufio.NewReader(bytes.NewReader(raw))
	var magic string
	var w, h, maxVal int
	if _, err := fmt.Fscan(r, &magic, &w, &h, &maxVal); err != nil || magic != "P5" {
		t.Fatalf("pgm: %q %v", magic, err)
	}
	if _, err := r.ReadByte(); err != nil {
		t.Fatal(err)
	}
	pixels := make([]byte, w*h)
	if _, err := io.ReadFull(r, pixels); err != nil {
		t.Fatal(err)
	}
	return w, h, pixels
}

func scannedPDF(t *testing.T, pages int) []byte {
	t.Helper()
	return imagePDF(t, pages, "q 612 0 0 300 0 0 cm /Im1 Do Q")
}

func imagePDF(t *testing.T, pages int, placement string) []byte {
	t.Helper()
	w, h, pixels := grayPixels(t, textPDF(scannedLine))
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	if _, err := zw.Write(pixels); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	image := fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream",
		w, h, z.Len(), z.String())
	kids := make([]string, pages)
	objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", "", image, stream(placement)}
	for i := range pages {
		kids[i] = fmt.Sprintf("%d 0 R", len(objects)+1)
		objects = append(objects, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 300] /Contents 4 0 R /Resources << /XObject << /Im1 3 0 R >> >> >>")
	}
	objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), pages)
	return pdfOf(objects)
}

func TestScannedPageIsReadByOCR(t *testing.T) {
	requireOCR(t)

	got, err := New().Extract(context.Background(), bytes.NewReader(scannedPDF(t, 1)), domain.SourceDiarioCamara)

	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Text, "DECRETO LEGISLATIVO") || !slices.Equal(got.OCRPages, []int{1}) {
		t.Errorf("OCR: %q %v", got.Text, got.OCRPages)
	}
}

func TestTextPagesAreNotReadByOCR(t *testing.T) {
	requireOCR(t)

	got, err := New().Extract(context.Background(), bytes.NewReader(twoPagePDF()), domain.SourceDiarioPrefeitura)

	if err != nil || len(got.OCRPages) != 0 || !strings.Contains(got.Text, "PAGINA UM") {
		t.Errorf("texto: %q %v %v", got.Text, got.OCRPages, err)
	}
}

func TestPageWithASmallImageIsNotReadByOCR(t *testing.T) {
	requireOCR(t)

	got, err := New().Extract(context.Background(), bytes.NewReader(imagePDF(t, 1, "q 180 0 0 90 0 0 cm /Im1 Do Q")), domain.SourceDiarioPrefeitura)

	if err != nil || len(got.OCRPages) != 0 {
		t.Errorf("foto pequena foi para o OCR: %v %v", got.OCRPages, err)
	}
}

func TestOCRStopsWhenItsTimeBudgetIsSpent(t *testing.T) {
	requireOCR(t)
	start := time.Now()
	calls := 0
	p := New()
	p.OCRBudget = time.Minute
	p.clock = func() time.Time {
		calls++
		if calls > 2 {
			return start.Add(2 * time.Minute)
		}
		return start
	}

	got, err := p.Extract(context.Background(), bytes.NewReader(scannedPDF(t, 2)), domain.SourceDiarioPrefeitura)

	if err != nil || !slices.Equal(got.OCRPages, []int{1}) {
		t.Errorf("limite: %v %v", got.OCRPages, err)
	}
	pages := strings.Split(got.Text, "\f")
	if len(pages) < 2 || !strings.Contains(pages[0], "DECRETO") || strings.Contains(pages[1], "DECRETO") {
		t.Errorf("páginas: %q", pages)
	}
}

func TestExtractPageReadsAScannedPageByOCR(t *testing.T) {
	requireOCR(t)

	got, pages, err := New().ExtractPage(context.Background(), bytes.NewReader(scannedPDF(t, 2)), domain.SourceDiarioCamara, 2)

	if err != nil || pages != 2 || !strings.Contains(got.Text, "DECRETO LEGISLATIVO") || !slices.Equal(got.OCRPages, []int{2}) {
		t.Errorf("página 2: %q %d %v %v", got.Text, pages, got.OCRPages, err)
	}
}

func TestPrefeituraPagesWithAPhotoAreNotReadByOCR(t *testing.T) {
	requirePoppler(t)
	for _, name := range []string{"2026_03_31.pdf", "2026_06_30.pdf", "2026_09_16.pdf"} {
		f, err := os.Open(filepath.Join("..", "..", "..", "testdata", "editions", name))
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("sem edições reais em testdata/editions (rode ./scripts/fetch-editions.sh)")
		}
		if err != nil {
			t.Fatal(err)
		}
		got, err := New().Extract(context.Background(), f, domain.SourceDiarioPrefeitura)
		f.Close()

		if err != nil || len(got.OCRPages) != 0 {
			t.Errorf("%s: %v %v", name, got.OCRPages, err)
		}
	}
}
