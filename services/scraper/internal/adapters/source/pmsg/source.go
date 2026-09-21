// Package pmsg implementa ports.EditionSource para o site do Diário Oficial
// da Prefeitura de São Gonçalo (https://do.pmsg.rj.gov.br/).
//
// Estrutura real do site (docs/parser-findings.md):
//   - os PDFs ficam em URLs determinísticas: diario/AAAA_MM_DD.pdf (200 quando
//     há edição, 500 quando não há);
//   - a "Busca Específica" (POST index com DataInicial, DataFinal, Termo) lista
//     as edições do período que contêm o termo, 5 por página, com links
//     href="diario/AAAA_MM_DD.pdf" e paginação por ?NumeroPagina=N.
//
// Listamos com um termo presente no cabeçalho de toda página ("Gonçalo"),
// percorremos a paginação e montamos as URLs. O site não exige JavaScript.
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
	userAgent = "diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)"
	// Termo que aparece no cabeçalho de todas as páginas de toda edição.
	listingTerm  = "Gonçalo"
	maxPages     = 50
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
	return &Source{baseURL: u, http: &http.Client{Timeout: 60 * time.Second}, delay: 2 * time.Second}, nil
}

var (
	editionLinkRe = regexp.MustCompile(`href="diario/(\d{4})_(\d{2})_(\d{2})\.pdf"`)
	pageLinkRe    = regexp.MustCompile(`NumeroPagina=(\d+)`)
)

// ListEditions consulta a busca do site para o período e devolve uma edição
// por data encontrada, da mais antiga para a mais recente.
func (s *Source) ListEditions(ctx context.Context, from, to time.Time) ([]domain.Edition, error) {
	from, to = dayStart(from), dayStart(to)
	seen := map[time.Time]bool{}
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
		dates, pages := parseListing(html)
		for _, d := range dates {
			if !d.Before(from) && !d.After(to) {
				seen[d] = true
			}
		}
		for _, p := range pages {
			if !visited[p] {
				pending = append(pending, p)
			}
		}
	}

	out := make([]domain.Edition, 0, len(seen))
	for d := range seen {
		out = append(out, domain.Edition{PublishedAt: d, URL: EditionURL(s.baseURL, d)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PublishedAt.Before(out[j].PublishedAt) })
	return out, nil
}

// listingPage faz a busca (POST na primeira página, GET nas seguintes, como
// o paginador do site) e devolve o HTML.
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

// parseListing extrai as datas das edições e os números das outras páginas.
func parseListing(html string) (dates []time.Time, pages []int) {
	seenDate := map[time.Time]bool{}
	for _, m := range editionLinkRe.FindAllStringSubmatch(html, -1) {
		d, err := time.Parse(time.DateOnly, m[1]+"-"+m[2]+"-"+m[3])
		if err != nil || seenDate[d] {
			continue
		}
		seenDate[d] = true
		dates = append(dates, d)
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
	return dates, pages
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

// do executa uma requisição respeitando a pausa entre chamadas (o caso de
// uso é sequencial: nunca há mais de uma requisição em andamento).
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
