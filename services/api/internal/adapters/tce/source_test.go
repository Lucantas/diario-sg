package tce

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const csvBody = "\xef\xbb\xbfEnte;Unidade;Ano;Mes;NumeroEmpenho;TipoPessoa;CPFCNPJ;Funcao;Empenhado;Liquidado;Pago\r\n" +
	"SAO GONCALO;PREFEITURA SÃO GONÇALO;2025;01;1;JURÍDICA;39818737000151;SAÚDE;10.5;0.0;0.0\r\n"

func newTCE(t *testing.T, handler http.HandlerFunc) *Source {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	s := New(srv.URL+"/api/v1/", srv.Client())
	s.retryWait = time.Millisecond
	return s
}

func TestCommitmentsReadsTheYearCSV(t *testing.T) {
	var query string
	src := newTCE(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Path + "?" + r.URL.RawQuery
		_, _ = w.Write([]byte(csvBody))
	})
	var rows [][]string
	var header []string

	sum, err := src.Commitments(context.Background(), 2025, func(h, r []string) error {
		header, rows = h, append(rows, r)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(query, "/api/v1/empenho_municipio?") || !strings.Contains(query, "ano=2025") || !strings.Contains(query, "municipio=SAO+GONCALO") || !strings.Contains(query, "csv=true") {
		t.Errorf("consulta: %s", query)
	}
	if len(header) != 11 || len(rows) != 1 || rows[0][1] != "PREFEITURA SÃO GONÇALO" {
		t.Errorf("cabeçalho %q linhas %q", header, rows)
	}
	want := sha256.Sum256([]byte(csvBody))
	if sum != hex.EncodeToString(want[:]) {
		t.Errorf("sha: %s", sum)
	}
}

func TestCommitmentsRetriesServerErrors(t *testing.T) {
	var calls atomic.Int32
	src := newTCE(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		_, _ = w.Write([]byte(csvBody))
	})

	if _, err := src.Commitments(context.Background(), 2025, func(_, _ []string) error { return nil }); err != nil {
		t.Fatalf("duas falhas deveriam ser absorvidas: %v", err)
	}
}

func TestCommitmentsDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	src := newTCE(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	})

	_, err := src.Commitments(context.Background(), 2025, func(_, _ []string) error { return nil })

	if err == nil || !strings.Contains(err.Error(), "empenhos de 2025") || calls.Load() != 1 {
		t.Fatalf("erro %v, chamadas %d", err, calls.Load())
	}
}
