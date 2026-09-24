package pdf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type PDFToText struct{ Timeout time.Duration }

func New() PDFToText { return PDFToText{Timeout: 2 * time.Minute} }

func (p PDFToText) Extract(ctx context.Context, r io.Reader, source string) (string, error) {
	var text string
	err := p.withFile(ctx, r, func(ctx context.Context, path string) error {
		var err error
		text, err = extract(ctx, path, source)
		return err
	})
	return text, err
}

func (p PDFToText) ExtractPage(ctx context.Context, r io.Reader, source string, page int) (string, int, error) {
	var text string
	var pages int
	err := p.withFile(ctx, r, func(ctx context.Context, path string) error {
		var err error
		if pages, err = pageCount(ctx, path); err != nil {
			return err
		}
		if page < 1 || page > pages {
			return fmt.Errorf("página %d de %d: %w", page, pages, domain.ErrInvalidInput)
		}
		n := strconv.Itoa(page)
		text, err = extract(ctx, path, source, "-f", n, "-l", n)
		return err
	})
	return text, pages, err
}

func (p PDFToText) withFile(ctx context.Context, r io.Reader, use func(context.Context, string) error) error {
	tmp, err := os.CreateTemp("", "gazette-*.pdf")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	return use(ctx, tmp.Name())
}

func extract(ctx context.Context, path, source string, pageFlags ...string) (string, error) {
	raw, err := pdftotext(ctx, slices.Concat(pageFlags, []string{path, "-"})...)
	if err != nil || source != domain.SourceDiarioCamara {
		return raw, err
	}
	layout, err := pdftotext(ctx, slices.Concat(pageFlags, []string{"-layout", path, "-"})...)
	if err != nil {
		return "", err
	}
	return mergePages(raw, layout), nil
}

var pagesRe = regexp.MustCompile(`(?m)^Pages:\s+(\d+)`)

func pageCount(ctx context.Context, path string) (int, error) {
	info, err := exec.CommandContext(ctx, "pdfinfo", path).Output()
	if err != nil {
		return 0, fmt.Errorf("pdfinfo: %w", err)
	}
	m := pagesRe.FindSubmatch(info)
	if m == nil {
		return 0, fmt.Errorf("pdfinfo sem o número de páginas")
	}
	return strconv.Atoi(string(m[1]))
}

func pdftotext(ctx context.Context, args ...string) (string, error) {
	var out, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "pdftotext", append([]string{"-enc", "UTF-8"}, args...)...)
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %w: %s", err, stderr.String())
	}
	return out.String(), nil
}
