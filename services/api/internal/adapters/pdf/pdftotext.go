package pdf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type PDFToText struct{ Timeout time.Duration }

func New() PDFToText { return PDFToText{Timeout: 2 * time.Minute} }

func (p PDFToText) Extract(ctx context.Context, r io.Reader, source string) (string, error) {
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
	raw, err := run(ctx, tmp.Name())
	if err != nil || source != domain.SourceDiarioCamara {
		return raw, err
	}
	layout, err := run(ctx, tmp.Name(), "-layout")
	if err != nil {
		return "", err
	}
	return mergePages(raw, layout), nil
}

func run(ctx context.Context, path string, flags ...string) (string, error) {
	args := append(append([]string{"-enc", "UTF-8"}, flags...), path, "-")
	var out, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "pdftotext", args...)
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %w: %s", err, stderr.String())
	}
	return out.String(), nil
}
