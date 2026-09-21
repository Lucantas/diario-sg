// Package pmsg implementa ports.EditionSource para o site do Diário Oficial
// da Prefeitura de São Gonçalo (https://do.pmsg.rj.gov.br/).
//
// ATENÇÃO: a extração abaixo é genérica (procura links para PDF e datas no
// texto/URL). Antes de ir para produção, inspecione o HTML real do site e
// ajuste ListEditions/parseListing. Veja também o spider proposto no Querido
// Diário: https://github.com/okfn-brasil/querido-diario/issues/1210
package pmsg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

const userAgent = "diario-sg-bot/0.1 (+https://github.com/seu-usuario/diario-sg)"

type Source struct {
	baseURL *url.URL
	http    *http.Client
	delay   time.Duration // pausa entre downloads: seja gentil com o site
}

func New(baseURL string) (*Source, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("SOURCE_URL inválida: %w", err)
	}
	return &Source{baseURL: u, http: &http.Client{Timeout: 60 * time.Second}, delay: 2 * time.Second}, nil
}

var (
	linkRe = regexp.MustCompile(`(?is)<a[^>]+href="([^"]+\.pdf)"[^>]*>(.*?)</a>`)
	dateBR = regexp.MustCompile(`(\d{2})[/.-](\d{2})[/.-](\d{4})`)
	dateIS = regexp.MustCompile(`(\d{4})[/.-](\d{2})[/.-](\d{2})`)
	numRe  = regexp.MustCompile(`(?i)edi(?:ç|c)(?:ã|a)o\s*(?:n[º°o.]*\s*)?(\d+)`)
	tagRe  = regexp.MustCompile(`<[^>]+>`)
)

func (s *Source) ListEditions(ctx context.Context, from, to time.Time) ([]domain.Edition, error) {
	// TODO(ajustar): se o site paginar ou filtrar por data via querystring,
	// itere aqui sobre as páginas/datas entre from e to.
	body, err := s.get(ctx, s.baseURL.String())
	if err != nil {
		return nil, err
	}
	defer body.Close()
	html, err := io.ReadAll(io.LimitReader(body, 10<<20))
	if err != nil {
		return nil, err
	}

	var out []domain.Edition
	for _, e := range s.parseListing(string(html)) {
		if !e.PublishedAt.Before(dayStart(from)) && !e.PublishedAt.After(to) {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *Source) parseListing(html string) []domain.Edition {
	var out []domain.Edition
	for _, m := range linkRe.FindAllStringSubmatch(html, -1) {
		href, text := m[1], stripTags(m[2])
		abs, err := s.baseURL.Parse(href)
		if err != nil {
			continue
		}
		published, ok := findDate(text + " " + href)
		if !ok {
			continue
		}
		e := domain.Edition{PublishedAt: published, URL: abs.String()}
		if n := numRe.FindStringSubmatch(text); n != nil {
			e.Number = n[1]
		}
		out = append(out, e)
	}
	return out
}

func (s *Source) Download(ctx context.Context, e domain.Edition) (io.ReadCloser, error) {
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return s.get(ctx, e.URL)
}

func (s *Source) get(ctx context.Context, u string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: status %d", u, resp.StatusCode)
	}
	return resp.Body, nil
}

func stripTags(s string) string { return strings.TrimSpace(tagRe.ReplaceAllString(s, " ")) }

func findDate(s string) (time.Time, bool) {
	if m := dateBR.FindStringSubmatch(s); m != nil {
		if t, err := time.Parse("02/01/2006", m[1]+"/"+m[2]+"/"+m[3]); err == nil {
			return t, true
		}
	}
	if m := dateIS.FindStringSubmatch(s); m != nil {
		if t, err := time.Parse("2006-01-02", m[1]+"-"+m[2]+"-"+m[3]); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
