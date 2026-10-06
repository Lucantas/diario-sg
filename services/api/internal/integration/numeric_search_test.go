//go:build integration

package integration

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

const numericGazette = `EXTRATO DO CONTRATO Nº 55/2026
Processo nº65/100410/2018. Contratada: Construtora Alfa LTDA, CNPJ 28.636.579/0001-00.
PORTARIA Nº 9/2026
Nomeia servidor da Secretaria de Obras.`

func TestSearchFindsNumbersInsideTextAndDropsWordFragments(t *testing.T) {
	srv, _ := newServerFor(t, numericGazette)
	cases := []struct {
		name, q string
		want    int
	}{
		{"processo parcial colado em outro texto", "100410/2018", 1},
		{"cnpj formatado", "28.636.579/0001-00", 1},
		{"percentual no meio do número é literal", "100%410", 0},
		{"texto sem dígito pelo full-text", "construtora", 1},
		{"trecho de palavra sem dígito não casa mais", "nstrutora alf", 0},
		{"texto com número colado em barra", "Portaria nº 9", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var res struct {
				Total int `json:"total"`
			}
			getJSON(t, srv.URL+"/v1/acts?q="+url.QueryEscape(c.q), &res)
			if res.Total != c.want {
				t.Errorf("%q: veio %d atos, queria %d", c.q, res.Total, c.want)
			}
		})
	}
}

func TestDigitSearchUsesNumericTermsIndex(t *testing.T) {
	_, db := newServerFor(t, numericGazette)
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SET enable_seqscan = off`); err != nil {
		t.Fatal(err)
	}
	rows, err := conn.QueryContext(ctx, `EXPLAIN SELECT id FROM acts WHERE numeric_terms ILIKE '%100410/2018%'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		plan.WriteString(line + "\n")
	}
	if !strings.Contains(plan.String(), "acts_numeric_terms_trgm_idx") {
		t.Fatalf("termo com dígito deveria usar o índice de numeric_terms:\n%s", plan.String())
	}
}
