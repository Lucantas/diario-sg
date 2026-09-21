// Package pdf extrai texto de PDFs usando o pdftotext (poppler-utils),
// instalado na imagem do container.
package pdf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

type PDFToText struct{ Timeout time.Duration }

func New() PDFToText { return PDFToText{Timeout: 2 * time.Minute} }

func (p PDFToText) Extract(ctx context.Context, r io.Reader) (string, error) {
	tmp, err := os.CreateTemp("", "gazette-*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	var out, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-enc", "UTF-8", tmp.Name(), "-")
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %w: %s", err, stderr.String())
	}
	// Se vier vazio, o PDF provavelmente é imagem escaneada: plugar OCR aqui.
	return out.String(), nil
}
