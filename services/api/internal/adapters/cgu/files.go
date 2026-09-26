package cgu

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"time"
	"unicode/utf8"
)

const monthLayout = "200601"

var publishedMonthRe = regexp.MustCompile(`"ano"\s*:\s*"(\d{4})",\s*"mes"\s*:\s*"(\d{2})"`)

func (s *Source) LatestMonth(ctx context.Context, dataset string) (time.Time, error) {
	page, err := s.get(ctx, s.baseURL+dataset)
	if err != nil {
		return time.Time{}, fmt.Errorf("página de %s: %w", dataset, err)
	}
	var latest time.Time
	for _, m := range publishedMonthRe.FindAllSubmatch(page, -1) {
		month, err := time.Parse(monthLayout, string(m[1])+string(m[2]))
		if err == nil && month.After(latest) {
			latest = month
		}
	}
	if latest.IsZero() {
		return time.Time{}, fmt.Errorf("página de %s sem mês de arquivo", dataset)
	}
	return latest, nil
}

func (s *Source) ZipCSVs(ctx context.Context, file string, each func(name string, header, row []string) error) (string, error) {
	body, err := s.get(ctx, s.baseURL+file)
	if err != nil {
		return "", fmt.Errorf("baixar %s: %w", file, err)
	}
	sum := sha256.Sum256(body)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return "", fmt.Errorf("abrir %s: %w", file, err)
	}
	for _, f := range zr.File {
		name := path.Base(f.Name)
		if err := streamCSV(f, func(header, row []string) error { return each(name, header, row) }); err != nil {
			return "", fmt.Errorf("ler %s de %s: %w", name, file, err)
		}
	}
	return hex.EncodeToString(sum[:]), nil
}

func streamCSV(f *zip.File, each func(header, row []string) error) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	r := csv.NewReader(&latin1Reader{src: bufio.NewReader(rc)})
	r.Comma, r.LazyQuotes, r.FieldsPerRecord = ';', true, -1
	header, err := r.Read()
	if err != nil {
		return fmt.Errorf("cabeçalho: %w", err)
	}
	for {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := each(header, row); err != nil {
			return err
		}
	}
}

type latin1Reader struct {
	src     *bufio.Reader
	pending []byte
}

func (l *latin1Reader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if len(l.pending) > 0 {
			c := copy(p[n:], l.pending)
			l.pending = l.pending[c:]
			n += c
			continue
		}
		b, err := l.src.ReadByte()
		if err != nil {
			if n > 0 {
				return n, nil
			}
			return 0, err
		}
		if b < utf8.RuneSelf {
			p[n] = b
			n++
			continue
		}
		l.pending = utf8.AppendRune(l.pending[:0], rune(b))
	}
	return n, nil
}
