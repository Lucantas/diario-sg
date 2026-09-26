package pdf

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const (
	maxScanLetters   = 200
	minImageCoverage = 0.25
	pointsPerInch    = 72
	ocrResolutionDPI = "300"
	ocrLanguage      = "por"
	ocrLayoutMode    = "1"
)

var pageSizeRe = regexp.MustCompile(`(?m)^Page\s+(\d+) size:\s+([\d.]+) x ([\d.]+)`)

func (p PDFToText) readScannedPages(ctx context.Context, path, text string, firstPage int) (map[int]string, error) {
	candidates, err := scannedPages(ctx, path, text, firstPage)
	if err != nil || len(candidates) == 0 {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "ocr-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	start := p.now()
	out := make(map[int]string, len(candidates))
	for _, page := range candidates {
		if p.OCRBudget > 0 && p.now().Sub(start) >= p.OCRBudget {
			break
		}
		read, err := ocrPage(ctx, path, dir, page)
		if err != nil {
			return nil, fmt.Errorf("OCR da página %d: %w", page, err)
		}
		out[page] = read
	}
	return out, nil
}

func scannedPages(ctx context.Context, path, text string, firstPage int) ([]int, error) {
	pages := strings.Split(text, "\f")
	var sparse []int
	for i, page := range pages {
		if countLetters(page) < maxScanLetters {
			sparse = append(sparse, firstPage+i)
		}
	}
	if len(sparse) == 0 {
		return nil, nil
	}
	pageRange := []string{"-f", strconv.Itoa(firstPage), "-l", strconv.Itoa(firstPage + len(pages) - 1)}
	coverage, err := imageCoverage(ctx, path, pageRange)
	if err != nil {
		return nil, err
	}
	var out []int
	for _, page := range sparse {
		if coverage[page] >= minImageCoverage {
			out = append(out, page)
		}
	}
	return out, nil
}

func countLetters(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}

func imageCoverage(ctx context.Context, path string, pageRange []string) (map[int]float64, error) {
	info, err := run(ctx, "pdfinfo", slices.Concat(pageRange, []string{path})...)
	if err != nil {
		return nil, err
	}
	areas := map[int]float64{}
	for _, m := range pageSizeRe.FindAllStringSubmatch(string(info), -1) {
		page, _ := strconv.Atoi(m[1])
		width, _ := strconv.ParseFloat(m[2], 64)
		height, _ := strconv.ParseFloat(m[3], 64)
		areas[page] = width * height
	}
	list, err := run(ctx, "pdfimages", slices.Concat(pageRange, []string{"-list", path})...)
	if err != nil {
		return nil, err
	}
	coverage := map[int]float64{}
	sc := bufio.NewScanner(bytes.NewReader(list))
	for sc.Scan() {
		page, area, ok := placedImageArea(strings.Fields(sc.Text()))
		if ok && areas[page] > 0 {
			coverage[page] += area / areas[page]
		}
	}
	return coverage, sc.Err()
}

func placedImageArea(f []string) (int, float64, bool) {
	if len(f) < 6 || f[2] != "image" {
		return 0, 0, false
	}
	page, errPage := strconv.Atoi(f[0])
	width, errWidth := strconv.ParseFloat(f[3], 64)
	height, errHeight := strconv.ParseFloat(f[4], 64)
	xPPI, errX := strconv.ParseFloat(f[len(f)-4], 64)
	yPPI, errY := strconv.ParseFloat(f[len(f)-3], 64)
	if errors.Join(errPage, errWidth, errHeight, errX, errY) != nil || xPPI <= 0 || yPPI <= 0 {
		return 0, 0, false
	}
	return page, width / xPPI * pointsPerInch * height / yPPI * pointsPerInch, true
}

func ocrPage(ctx context.Context, path, dir string, page int) (string, error) {
	n := strconv.Itoa(page)
	prefix := filepath.Join(dir, "p"+n)
	if _, err := run(ctx, "pdftoppm", "-f", n, "-l", n, "-r", ocrResolutionDPI, "-gray", "-png", "-singlefile", path, prefix); err != nil {
		return "", err
	}
	out, err := runSingleThreaded(ctx, "tesseract", prefix+".png", "stdout", "-l", ocrLanguage, "--psm", ocrLayoutMode)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(out), "\f", ""), nil
}

func run(ctx context.Context, bin string, args ...string) ([]byte, error) {
	return runCommand(exec.CommandContext(ctx, bin, args...))
}

func runSingleThreaded(ctx context.Context, bin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(), "OMP_THREAD_LIMIT=1")
	return runCommand(cmd)
}

func runCommand(cmd *exec.Cmd) ([]byte, error) {
	var out, stderr bytes.Buffer
	bin := filepath.Base(cmd.Path)
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w: %s", bin, err, strings.TrimSpace(stderr.String()))
	}
	return out.Bytes(), nil
}

func replacePages(text string, read map[int]string, firstPage int) string {
	if len(read) == 0 {
		return text
	}
	pages := strings.Split(text, "\f")
	for page, content := range read {
		if i := page - firstPage; i >= 0 && i < len(pages) {
			pages[i] = content
		}
	}
	return strings.Join(pages, "\f")
}

func sortedPages(read map[int]string) []int {
	out := make([]int, 0, len(read))
	for page := range read {
		out = append(out, page)
	}
	slices.Sort(out)
	return out
}
