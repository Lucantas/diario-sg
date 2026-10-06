package sourcepdf

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	userAgent       = "diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)"
	defaultMaxBytes = 100 << 20
	maxRedirects    = 5
)

var errTooLarge = errors.New("pdf oficial acima do tamanho máximo")

type Fetcher struct {
	http     *http.Client
	maxBytes int64
}

func New() *Fetcher {
	return &Fetcher{
		http:     &http.Client{Timeout: 2 * time.Minute, CheckRedirect: sameHostRedirect},
		maxBytes: defaultMaxBytes,
	}
}

func sameHostRedirect(req *http.Request, via []*http.Request) error {
	first := via[0].URL
	if len(via) >= maxRedirects {
		return fmt.Errorf("pdf oficial: redirects demais")
	}
	if req.URL.Host != first.Host || (first.Scheme == "https" && req.URL.Scheme != "https") {
		return fmt.Errorf("pdf oficial: redirect para fora de %s recusado", first.Host)
	}
	return nil
}

func (f *Fetcher) Open(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("pdf oficial %s: status %d", url, resp.StatusCode)
	}
	br := bufio.NewReader(&limitedReader{r: resp.Body, left: f.maxBytes})
	head, _ := br.Peek(5)
	if !bytes.HasPrefix(head, []byte("%PDF-")) {
		resp.Body.Close()
		return nil, fmt.Errorf("pdf oficial %s: resposta não é PDF", url)
	}
	return struct {
		io.Reader
		io.Closer
	}{br, resp.Body}, nil
}

type limitedReader struct {
	r    io.Reader
	left int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.left <= 0 {
		var one [1]byte
		if n, _ := l.r.Read(one[:]); n > 0 {
			return 0, errTooLarge
		}
		return 0, io.EOF
	}
	if int64(len(p)) > l.left {
		p = p[:l.left]
	}
	n, err := l.r.Read(p)
	l.left -= int64(n)
	return n, err
}
