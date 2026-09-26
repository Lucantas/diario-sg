package agentes

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxResponseBytes = 64 << 20
	defaultPause     = time.Second
	prefeituraEntity = "1"
	wholeYear        = "00"
	exportFormat     = "JSON"
	formField        = "ctl00$containerCorpo$"
)

var hiddenFieldRe = regexp.MustCompile(`<input[^>]*name="(__[A-Z]+)"[^>]*value="([^"]*)"`)

type Source struct {
	prefeituraURL string
	camaraURL     string
	sicamURL      string
	client        *http.Client
	pause         time.Duration

	mu   sync.Mutex
	last time.Time
}

func New(prefeituraURL, camaraURL, sicamURL string, client *http.Client) *Source {
	return &Source{prefeituraURL: prefeituraURL, camaraURL: camaraURL, sicamURL: sicamURL, client: client, pause: defaultPause}
}

func (s *Source) PrefeituraPay(ctx context.Context, year, month int) ([]byte, error) {
	q := url.Values{"flag": {"remuneracao"}, "entidade": {prefeituraEntity}, "competencia": {fmt.Sprintf("%02d/%d", month, year)}}
	body, err := s.do(ctx, s.client, http.MethodGet, s.prefeituraURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("folha da Prefeitura de %02d/%d: %w", month, year, err)
	}
	return body, nil
}

func (s *Source) CamaraPay(ctx context.Context, year int) ([]byte, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := *s.client
	client.Jar = jar
	page, err := s.do(ctx, &client, http.MethodGet, s.camaraURL, nil)
	if err != nil {
		return nil, fmt.Errorf("formulário da folha da Câmara: %w", err)
	}
	form := url.Values{"__EVENTTARGET": {""}, "__EVENTARGUMENT": {""}}
	for _, m := range hiddenFieldRe.FindAllSubmatch(page, -1) {
		form.Set(string(m[1]), html.UnescapeString(string(m[2])))
	}
	if form.Get("__VIEWSTATE") == "" {
		return nil, fmt.Errorf("formulário da folha da Câmara sem __VIEWSTATE")
	}
	form.Set(formField+"cbxAno", strconv.Itoa(year))
	form.Set(formField+"cbxMes", wholeYear)
	form.Set(formField+"cbxFormato", exportFormat)
	form.Set(formField+"btnAplicFiltro", "Exportar")
	body, err := s.do(ctx, &client, http.MethodPost, s.camaraURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("folha da Câmara de %d: %w", year, err)
	}
	if first := bytes.TrimSpace(body); len(first) == 0 || (first[0] != '[' && first[0] != '{') {
		return nil, fmt.Errorf("folha da Câmara de %d: a exportação não devolveu JSON", year)
	}
	return body, nil
}

func (s *Source) Councillors(ctx context.Context, year int) ([]byte, error) {
	body, err := s.do(ctx, s.client, http.MethodGet, s.sicamURL+"?Parlamentares/"+strconv.Itoa(year)+"/json", nil)
	if err != nil {
		return nil, fmt.Errorf("vereadores de %d no SICAM: %w", year, err)
	}
	return body, nil
}

func (s *Source) do(ctx context.Context, client *http.Client, method, u string, form io.Reader) ([]byte, error) {
	if err := s.wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u, form)
	if err != nil {
		return nil, err
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("resposta maior que %d bytes", maxResponseBytes)
	}
	return body, nil
}

func (s *Source) wait(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if delay := s.pause - time.Since(s.last); delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	s.last = time.Now()
	return nil
}
