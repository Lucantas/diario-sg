package pncp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const (
	pageSize          = 50
	attempts          = 4
	maxBodyBytes      = 16 << 20
	defaultPause      = 3 * time.Second
	defaultLimitWait  = time.Minute
	companySupplier   = "PJ"
	pncpDateLayout    = "2006-01-02"
	pncpDateTimeShort = 10
)

type Source struct {
	baseURL   string
	client    *http.Client
	pause     time.Duration
	limitWait time.Duration
}

func New(baseURL string, client *http.Client) *Source {
	return &Source{baseURL: baseURL, client: client, pause: defaultPause, limitWait: defaultLimitWait}
}

type page struct {
	Data             []contract `json:"data"`
	PaginasRestantes int        `json:"paginasRestantes"`
}

type contract struct {
	NumeroControlePNCP string  `json:"numeroControlePNCP"`
	AnoContrato        int     `json:"anoContrato"`
	SequencialContrato int     `json:"sequencialContrato"`
	Processo           string  `json:"processo"`
	NumeroContrato     string  `json:"numeroContratoEmpenho"`
	TipoPessoa         string  `json:"tipoPessoa"`
	NiFornecedor       string  `json:"niFornecedor"`
	NomeFornecedor     string  `json:"nomeRazaoSocialFornecedor"`
	Objeto             string  `json:"objetoContrato"`
	ValorGlobal        float64 `json:"valorGlobal"`
	DataAssinatura     string  `json:"dataAssinatura"`
	DataPublicacao     string  `json:"dataPublicacaoPncp"`
	VigenciaInicio     string  `json:"dataVigenciaInicio"`
	VigenciaFim        string  `json:"dataVigenciaFim"`
	TipoContrato       struct {
		Nome string `json:"nome"`
	} `json:"tipoContrato"`
	OrgaoEntidade struct {
		CNPJ string `json:"cnpj"`
	} `json:"orgaoEntidade"`
	UnidadeOrgao struct {
		Nome string `json:"nomeUnidade"`
	} `json:"unidadeOrgao"`
}

func (s *Source) Contracts(ctx context.Context, org string, year int, each func(c domain.PNCPContract, company bool) error) (string, error) {
	h := sha256.New()
	for n := 1; ; n++ {
		body, err := s.page(ctx, org, year, n)
		if err != nil {
			return "", fmt.Errorf("contratos de %s em %d, página %d: %w", org, year, n, err)
		}
		h.Write(body)
		if len(body) == 0 {
			break
		}
		var p page
		if err := json.Unmarshal(body, &p); err != nil {
			return "", fmt.Errorf("contratos de %s em %d, página %d: %w", org, year, n, err)
		}
		for _, c := range p.Data {
			if err := each(c.domain(), c.isCompany()); err != nil {
				return "", err
			}
		}
		if p.PaginasRestantes <= 0 {
			break
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (s *Source) page(ctx context.Context, org string, year, n int) ([]byte, error) {
	q := url.Values{"dataInicial": {fmt.Sprintf("%d0101", year)}, "dataFinal": {fmt.Sprintf("%d1231", year)},
		"cnpjOrgao": {org}, "pagina": {strconv.Itoa(n)}, "tamanhoPagina": {strconv.Itoa(pageSize)}}
	u := s.baseURL + "contratos?" + q.Encode()
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := wait(ctx, s.pause); err != nil {
			return nil, err
		}
		body, limited, err := s.get(ctx, u)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if limited {
			if err := wait(ctx, s.limitWait*time.Duration(attempt)); err != nil {
				return nil, err
			}
		}
	}
	return nil, lastErr
}

func (s *Source) get(ctx context.Context, u string) ([]byte, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, true, err
	}
	switch {
	case resp.StatusCode == http.StatusNoContent:
		return nil, false, nil
	case resp.StatusCode == http.StatusOK && !bytes.HasPrefix(bytes.TrimSpace(body), []byte("<")):
		return body, false, nil
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusOK:
		return nil, true, fmt.Errorf("status %d (limite de requisições ou erro do servidor)", resp.StatusCode)
	default:
		return nil, false, fmt.Errorf("status %d", resp.StatusCode)
	}
}

func wait(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func (c contract) isCompany() bool {
	cnpj, ok := domain.NormalizeCNPJ(c.NiFornecedor)
	return c.TipoPessoa == companySupplier && ok && domain.HasValidCheckDigits(cnpj)
}

func (c contract) domain() domain.PNCPContract {
	return domain.PNCPContract{ControlNumber: c.NumeroControlePNCP, OrgCNPJ: c.OrgaoEntidade.CNPJ, UnitName: c.UnidadeOrgao.Nome,
		Year: c.AnoContrato, Sequence: c.SequencialContrato, Kind: c.TipoContrato.Nome, Process: c.Processo, Number: c.NumeroContrato,
		SupplierCNPJ: supplierCNPJ(c.NiFornecedor), SupplierName: c.NomeFornecedor, Object: c.Objeto,
		ValueCents: int64(math.Round(c.ValorGlobal * 100)), SignedAt: date(c.DataAssinatura), PublishedAt: date(c.DataPublicacao),
		StartsAt: date(c.VigenciaInicio), EndsAt: date(c.VigenciaFim)}
}

func date(s string) *time.Time {
	if len(s) < pncpDateTimeShort {
		return nil
	}
	t, err := time.Parse(pncpDateLayout, s[:pncpDateTimeShort])
	if err != nil {
		return nil
	}
	return &t
}

func supplierCNPJ(ni string) string {
	if cnpj, ok := domain.NormalizeCNPJ(ni); ok {
		return cnpj
	}
	return ""
}
