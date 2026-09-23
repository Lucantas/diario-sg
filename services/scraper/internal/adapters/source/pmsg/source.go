package pmsg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

const (
	userAgent   = "diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)"
	listingTerm = "a"

	maxPages     = 200
	maxHTMLBytes = 10 << 20
)

type Source struct {
	baseURL *url.URL
	http    *http.Client
	delay   time.Duration
	last    time.Time
}

func New(baseURL string) (*Source, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("SOURCE_URL inválida: %q", baseURL)
	}
	return &Source{baseURL: u, http: newClient(), delay: 2 * time.Second}, nil
}

func newClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	transport.ResponseHeaderTimeout = 30 * time.Second
	return &http.Client{Timeout: 120 * time.Second, Transport: transport}
}

var (
	editionLinkRe = regexp.MustCompile(`href="(diario/(\d{4})_(\d{2})_(\d{2})(?:_\d+)?\.pdf)"`)
	pageLinkRe    = regexp.MustCompile(`NumeroPagina=(\d+)`)
)

type listedEdition struct {
	path string
	date time.Time
}

func (s *Source) ListEditions(ctx context.Context, from, to time.Time) ([]domain.Edition, error) {
	from, to = dayStart(from), dayStart(to)
	seen := map[string]time.Time{}
	pending := []int{1}
	visited := map[int]bool{}
	for len(pending) > 0 && len(visited) < maxPages {
		page := pending[0]
		pending = pending[1:]
		if visited[page] {
			continue
		}
		visited[page] = true

		html, err := s.listingPage(ctx, from, to, page)
		if err != nil {
			return nil, err
		}
		listed, pages := parseListing(html)
		for _, e := range listed {
			if !e.date.Before(from) && !e.date.After(to) {
				seen[e.path] = e.date
			}
		}
		for _, p := range pages {
			if !visited[p] {
				pending = append(pending, p)
			}
		}
	}

	if len(pending) > 0 {
		return nil, fmt.Errorf("listagem passou de %d páginas para %s..%s; reduza a janela",
			maxPages, from.Format(time.DateOnly), to.Format(time.DateOnly))
	}

	out := make([]domain.Edition, 0, len(seen))
	for path, d := range seen {
		out = append(out, domain.Edition{PublishedAt: d, URL: s.baseURL.ResolveReference(&url.URL{Path: path}).String()})
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].PublishedAt.Equal(out[j].PublishedAt) {
			return out[i].PublishedAt.Before(out[j].PublishedAt)
		}
		return out[i].URL < out[j].URL
	})
	return out, nil
}

func (s *Source) listingPage(ctx context.Context, from, to time.Time, page int) (string, error) {
	form := url.Values{
		"DataInicial":    {from.Format(time.DateOnly)},
		"DataFinal":      {to.Format(time.DateOnly)},
		"Termo":          {listingTerm},
		"PesquisarTermo": {"Pesquisar"},
	}
	var req *http.Request
	var err error
	target := s.baseURL.ResolveReference(&url.URL{Path: "index"})
	if page == 1 {
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, target.String(), strings.NewReader(form.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	} else {
		form.Set("NumeroPagina", strconv.Itoa(page))
		target.RawQuery = form.Encode()
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	}
	if err != nil {
		return "", err
	}
	body, err := s.do(ctx, req)
	if err != nil {
		return "", fmt.Errorf("listar página %d: %w", page, err)
	}
	defer body.Close()
	html, err := io.ReadAll(io.LimitReader(body, maxHTMLBytes))
	if err != nil {
		return "", err
	}
	return string(html), nil
}

func parseListing(html string) (editions []listedEdition, pages []int) {
	seenPath := map[string]bool{}
	for _, m := range editionLinkRe.FindAllStringSubmatch(html, -1) {
		d, err := time.Parse(time.DateOnly, m[2]+"-"+m[3]+"-"+m[4])
		if err != nil || seenPath[m[1]] {
			continue
		}
		seenPath[m[1]] = true
		editions = append(editions, listedEdition{path: m[1], date: d})
	}
	seenPage := map[int]bool{}
	for _, m := range pageLinkRe.FindAllStringSubmatch(html, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil || n < 1 || seenPage[n] {
			continue
		}
		seenPage[n] = true
		pages = append(pages, n)
	}
	return editions, pages
}

func (s *Source) Download(ctx context.Context, e domain.Edition) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.URL, nil)
	if err != nil {
		return nil, err
	}
	body, err := s.do(ctx, req)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (s *Source) do(ctx context.Context, req *http.Request) (io.ReadCloser, error) {
	if err := s.throttle(ctx); err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s %s: status %d", req.Method, req.URL, resp.StatusCode)
	}
	return resp.Body, nil
}

func (s *Source) throttle(ctx context.Context) error {
	wait := s.delay - time.Since(s.last)
	if !s.last.IsZero() && wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.last = time.Now()
	return nil
}

func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
