package receita

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	blockSize     = 16 << 20
	fetchAttempts = 3
)

type rangeReaderAt struct {
	ctx        context.Context
	src        *Source
	url        string
	size       int64
	block      []byte
	blockStart int64
}

func (r *rangeReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= r.size {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) && off+int64(n) < r.size {
		pos := off + int64(n)
		if pos < r.blockStart || pos >= r.blockStart+int64(len(r.block)) {
			if err := r.fill(pos); err != nil {
				return n, err
			}
		}
		n += copy(p[n:], r.block[pos-r.blockStart:])
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (r *rangeReaderAt) fill(start int64) error {
	end := min(start+blockSize, r.size) - 1
	var lastErr error
	for attempt := 0; attempt < fetchAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-r.ctx.Done():
				return r.ctx.Err()
			case <-time.After(r.src.retryWait * time.Duration(attempt)):
			}
		}
		body, err := r.fetch(start, end)
		if err == nil {
			r.block, r.blockStart = body, start
			return nil
		}
		lastErr = err
	}
	return fmt.Errorf("bytes %d-%d de %s: %w", start, end, r.url, lastErr)
}

func (r *rangeReaderAt) fetch(start, end int64) ([]byte, error) {
	req, err := r.src.request(r.ctx, http.MethodGet, r.url)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	resp, err := r.src.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) != end-start+1 {
		return nil, fmt.Errorf("resposta com %d bytes, esperava %d", len(body), end-start+1)
	}
	return body, nil
}
