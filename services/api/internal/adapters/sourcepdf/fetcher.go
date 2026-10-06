package sourcepdf

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const userAgent = "diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)"

type Fetcher struct{ http *http.Client }

func New() *Fetcher { return &Fetcher{http: &http.Client{Timeout: 2 * time.Minute}} }

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
	br := bufio.NewReader(resp.Body)
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
