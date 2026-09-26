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

type PDFToText struct {
	Timeout   time.Duration
	OCRBudget time.Duration
	clock     func() time.Time
}

const (
	defaultTimeout   = 4 * time.Minute
	defaultOCRBudget = 90 * time.Second
)

func New() PDFToText { return PDFToText{Timeout: defaultTimeout, OCRBudget: defaultOCRBudget} }

func (p PDFToText) now() time.Time {
	if p.clock == nil {
		return time.Now()
	}
	return p.clock()
}

func (p PDFToText) Extract(ctx context.Context, r io.Reader, source string) (domain.ExtractedText, error) {
	var text domain.ExtractedText
	err := p.withFile(ctx, r, func(ctx context.Context, path string) error {
		var err error
		text, err = p.extract(ctx, path, source, 1)
		return err
	})
	return text, err
}

func (p PDFToText) ExtractPage(ctx context.Context, r io.Reader, source string, page int) (domain.ExtractedText, int, error) {
	var text domain.ExtractedText
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
		text, err = p.extract(ctx, path, source, page, "-f", n, "-l", n)
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

func (p PDFToText) extract(ctx context.Context, path, source string, firstPage int, pageFlags ...string) (domain.ExtractedText, error) {
	raw, err := pdftotext(ctx, slices.Concat(pageFlags, []string{path, "-"})...)
	if err != nil {
		return domain.ExtractedText{}, err
	}
	scanned, err := p.readScannedPages(ctx, path, raw, firstPage)
	if err != nil {
		return domain.ExtractedText{}, err
	}
	raw = replacePages(raw, scanned, firstPage)
	if source == domain.SourceDiarioCamara {
		layout, err := pdftotext(ctx, slices.Concat(pageFlags, []string{"-layout", path, "-"})...)
		if err != nil {
			return domain.ExtractedText{}, err
		}
		raw = mergePages(raw, replacePages(layout, scanned, firstPage))
	}
	return domain.ExtractedText{Text: raw, OCRPages: sortedPages(scanned)}, nil
}

var pagesRe = regexp.MustCompile(`(?m)^Pages:\s+(\d+)`)

func pageCount(ctx context.Context, path string) (int, error) {
	var info, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "pdfinfo", path)
	cmd.Stdout, cmd.Stderr = &info, &stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("pdfinfo: %w: %s", err, stderr.String())
	}
	m := pagesRe.FindSubmatch(info.Bytes())
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
