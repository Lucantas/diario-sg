package receita

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const defaultRetryWait = 5 * time.Second

var monthDirRe = regexp.MustCompile(`^\d{4}-\d{2}$`)

type Source struct {
	baseURL   string
	token     string
	client    *http.Client
	retryWait time.Duration
}

func New(baseURL, token string, client *http.Client) *Source {
	return &Source{baseURL: strings.TrimSuffix(baseURL, "/") + "/", token: token, client: client, retryWait: defaultRetryWait}
}

func (s *Source) request(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.token, "")
	return req, nil
}

type multistatus struct {
	Responses []struct {
		Href string `xml:"href"`
	} `xml:"response"`
}

func (s *Source) list(ctx context.Context, dir string) ([]string, error) {
	req, err := s.request(ctx, "PROPFIND", s.baseURL+dir)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", "1")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listar %q: %w", dir, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("listar %q: status %d", dir, resp.StatusCode)
	}
	var ms multistatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, fmt.Errorf("listar %q: %w", dir, err)
	}
	names := make([]string, 0, len(ms.Responses))
	for _, r := range ms.Responses {
		if name := path.Base(strings.TrimSuffix(r.Href, "/")); name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

func (s *Source) LatestMonth(ctx context.Context) (string, error) {
	names, err := s.list(ctx, "")
	if err != nil {
		return "", err
	}
	latest := ""
	for _, n := range names {
		if monthDirRe.MatchString(n) && n > latest {
			latest = n
		}
	}
	if latest == "" {
		return "", errors.New("nenhum mês publicado no compartilhamento da Receita")
	}
	return latest, nil
}

func (s *Source) Files(ctx context.Context, month string) ([]string, error) {
	names, err := s.list(ctx, month+"/")
	if err != nil {
		return nil, err
	}
	var zips []string
	for _, n := range names {
		if strings.HasSuffix(n, ".zip") {
			zips = append(zips, n)
		}
	}
	sort.Strings(zips)
	return zips, nil
}

func (s *Source) Rows(ctx context.Context, month, file string, each func(fields []string) error) (string, error) {
	url := s.baseURL + month + "/" + file
	size, err := s.size(ctx, url)
	if err != nil {
		return "", err
	}
	zr, err := zip.NewReader(&rangeReaderAt{ctx: ctx, src: s, url: url, size: size}, size)
	if err != nil {
		return "", fmt.Errorf("abrir %s: %w", file, err)
	}
	if len(zr.File) != 1 {
		return "", fmt.Errorf("%s tem %d arquivos, esperava 1", file, len(zr.File))
	}
	rc, err := zr.File[0].Open()
	if err != nil {
		return "", fmt.Errorf("abrir %s: %w", file, err)
	}
	defer rc.Close()
	sum := sha256.New()
	r := csv.NewReader(&latin1Reader{r: bufio.NewReaderSize(io.TeeReader(rc, sum), 1<<20)})
	r.Comma, r.LazyQuotes, r.FieldsPerRecord, r.ReuseRecord = ';', true, -1, true
	for {
		fields, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("ler %s: %w", file, err)
		}
		if err := each(fields); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func (s *Source) size(ctx context.Context, url string) (int64, error) {
	req, err := s.request(ctx, http.MethodHead, url)
	if err != nil {
		return 0, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("tamanho de %s: %w", url, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength <= 0 {
		return 0, fmt.Errorf("tamanho de %s: status %d", url, resp.StatusCode)
	}
	return resp.ContentLength, nil
}

func (s *Source) Codes(ctx context.Context, month string) (domain.RegistryCodes, error) {
	c := domain.RegistryCodes{}
	for file, dst := range map[string]*map[string]string{
		"Cnaes.zip": &c.Activities, "Municipios.zip": &c.Cities, "Naturezas.zip": &c.Natures,
		"Qualificacoes.zip": &c.Roles, "Motivos.zip": &c.Reasons,
	} {
		table := map[string]string{}
		if _, err := s.Rows(ctx, month, file, func(f []string) error {
			if len(f) >= 2 {
				table[f[0]] = strings.TrimSpace(f[1])
			}
			return nil
		}); err != nil {
			return c, err
		}
		*dst = table
	}
	return c, nil
}

type latin1Reader struct {
	r   *bufio.Reader
	buf []byte
}

func (l *latin1Reader) Read(p []byte) (int, error) {
	if len(p) < utf8.UTFMax {
		return 0, io.ErrShortBuffer
	}
	want := len(p) / 2
	if cap(l.buf) < want {
		l.buf = make([]byte, want)
	}
	n, err := l.r.Read(l.buf[:want])
	out := 0
	for _, b := range l.buf[:n] {
		out += utf8.EncodeRune(p[out:], rune(b))
	}
	return out, err
}
