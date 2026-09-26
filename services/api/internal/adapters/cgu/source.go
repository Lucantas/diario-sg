package cgu

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	dayLayout          = "20060102"
	maxZipBytes        = 256 << 20
	defaultMinInterval = 20 * time.Second
)

var ErrHumanVerification = errors.New("o Portal da Transparência pediu verificação humana (AWS WAF); tente mais tarde")

var publishedDayRe = regexp.MustCompile(`"ano"\s*:\s*"(\d{4})",\s*"mes"\s*:\s*"(\d{2})",\s*"dia"\s*:\s*"(\d{2})"`)

type Source struct {
	baseURL     string
	client      *http.Client
	minInterval time.Duration
	last        time.Time
}

func New(baseURL string, client *http.Client) *Source {
	return &Source{baseURL: strings.TrimSuffix(baseURL, "/") + "/", client: client, minInterval: defaultMinInterval}
}

func (s *Source) throttle(ctx context.Context) error {
	wait := time.Until(s.last.Add(s.minInterval))
	if wait > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	s.last = time.Now()
	return nil
}

func (s *Source) get(ctx context.Context, url string) ([]byte, error) {
	if err := s.throttle(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusAccepted {
		return nil, ErrHumanVerification
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxZipBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxZipBytes {
		return nil, fmt.Errorf("resposta maior que %d bytes", maxZipBytes)
	}
	return body, nil
}

func (s *Source) LatestDay(ctx context.Context, register string) (time.Time, error) {
	page, err := s.get(ctx, s.baseURL+strings.ToLower(register))
	if err != nil {
		return time.Time{}, fmt.Errorf("página do %s: %w", register, err)
	}
	var latest time.Time
	for _, m := range publishedDayRe.FindAllSubmatch(page, -1) {
		day, err := time.Parse(dayLayout, string(m[1])+string(m[2])+string(m[3]))
		if err == nil && day.After(latest) {
			latest = day
		}
	}
	if latest.IsZero() {
		return time.Time{}, fmt.Errorf("página do %s sem data de arquivo", register)
	}
	return latest, nil
}

func (s *Source) Rows(ctx context.Context, register string, day time.Time, each func(header, row []string) error) (string, error) {
	name := fmt.Sprintf("%s de %s", register, day.Format(time.DateOnly))
	body, err := s.get(ctx, s.baseURL+strings.ToLower(register)+"/"+day.Format(dayLayout))
	if err != nil {
		return "", fmt.Errorf("baixar %s: %w", name, err)
	}
	sum := sha256.Sum256(body)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return "", fmt.Errorf("abrir %s: %w", name, err)
	}
	if len(zr.File) != 1 {
		return "", fmt.Errorf("%s com %d arquivos, esperava 1", name, len(zr.File))
	}
	if err := readCSV(zr.File[0], each); err != nil {
		return "", fmt.Errorf("ler %s: %w", name, err)
	}
	return hex.EncodeToString(sum[:]), nil
}

func readCSV(f *zip.File, each func(header, row []string) error) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	r := csv.NewReader(strings.NewReader(fromLatin1(raw)))
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

func fromLatin1(b []byte) string {
	out := make([]byte, 0, len(b)+len(b)/8)
	for _, c := range b {
		out = utf8.AppendRune(out, rune(c))
	}
	return string(out)
}
