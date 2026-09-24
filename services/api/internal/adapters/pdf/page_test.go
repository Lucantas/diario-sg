package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func twoPagePDF() []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R 5 0 R] /Count 2 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 200] /Contents 4 0 R /Resources << /Font << /F1 7 0 R >> >> >>",
		stream("BT /F1 12 Tf 20 100 Td (PAGINA UM) Tj ET"),
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 200] /Contents 6 0 R /Resources << /Font << /F1 7 0 R >> >> >>",
		stream("BT /F1 12 Tf 20 100 Td (PAGINA DOIS) Tj ET"),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
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

func stream(content string) string {
	return fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content)
}

func requirePoppler(t *testing.T) {
	t.Helper()
	for _, bin := range []string{"pdftotext", "pdfinfo"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s não instalado", bin)
		}
	}
}

func TestExtractPageReadsOnlyThatPage(t *testing.T) {
	requirePoppler(t)

	text, pages, err := New().ExtractPage(context.Background(), bytes.NewReader(twoPagePDF()), domain.SourceDiarioPrefeitura, 2)

	if err != nil || pages != 2 || !strings.Contains(text, "PAGINA DOIS") || strings.Contains(text, "PAGINA UM") {
		t.Fatalf("página 2 de 2: %q %d %v", text, pages, err)
	}
}

func TestExtractPageBeyondTheLastIsInvalid(t *testing.T) {
	requirePoppler(t)

	_, pages, err := New().ExtractPage(context.Background(), bytes.NewReader(twoPagePDF()), domain.SourceDiarioPrefeitura, 3)

	if !errors.Is(err, domain.ErrInvalidInput) || pages != 2 {
		t.Fatalf("página 3 de 2 deveria ser entrada inválida: %d %v", pages, err)
	}
}
