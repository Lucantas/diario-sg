package domain

import (
	"strings"
	"testing"
	"time"
)

const ceisHeader = `CADASTRO;CÓDIGO DA SANÇÃO;TIPO DE PESSOA;CPF OU CNPJ DO SANCIONADO;NOME DO SANCIONADO;NOME INFORMADO PELO ÓRGÃO SANCIONADOR;RAZÃO SOCIAL - CADASTRO RECEITA;NOME FANTASIA - CADASTRO RECEITA;NÚMERO DO PROCESSO;CATEGORIA DA SANÇÃO;DATA INÍCIO SANÇÃO;DATA FINAL SANÇÃO;DATA PUBLICAÇÃO;PUBLICAÇÃO;DETALHAMENTO DO MEIO DE PUBLICAÇÃO;DATA DO TRÂNSITO EM JULGADO;ABRAGÊNCIA DA SANÇÃO;ÓRGÃO SANCIONADOR;UF ÓRGÃO SANCIONADOR;ESFERA ÓRGÃO SANCIONADOR;FUNDAMENTAÇÃO LEGAL;DATA ORIGEM INFORMAÇÃO;ORIGEM INFORMAÇÕES;OBSERVAÇÕES`

const cnepHeader = `CADASTRO;CÓDIGO DA SANÇÃO;TIPO DE PESSOA;CPF OU CNPJ DO SANCIONADO;NOME DO SANCIONADO;NOME INFORMADO PELO ÓRGÃO SANCIONADOR;RAZÃO SOCIAL - CADASTRO RECEITA;NOME FANTASIA - CADASTRO RECEITA;NÚMERO DO PROCESSO;CATEGORIA DA SANÇÃO;VALOR DA MULTA;DATA INÍCIO SANÇÃO;DATA FINAL SANÇÃO;DATA PUBLICAÇÃO;PUBLICAÇÃO;DETALHAMENTO DO MEIO DE PUBLICAÇÃO;DATA DO TRÂNSITO EM JULGADO;ABRAGÊNCIA DA SANÇÃO;ÓRGÃO SANCIONADOR;UF ÓRGÃO SANCIONADOR;ESFERA ÓRGÃO SANCIONADOR;FUNDAMENTAÇÃO LEGAL;DATA ORIGEM INFORMAÇÃO;ORIGEM INFORMAÇÕES;OBSERVAÇÕES`

func columns(t *testing.T, header string) SanctionColumns {
	t.Helper()
	c, err := NewSanctionColumns(strings.Split(header, ";"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func ceisRow() []string {
	return strings.Split("CEIS;379001;J;28926250000176;EMPRESA X;Empresa X Ltda;EMPRESA X COMERCIO LTDA;X;SEI 123/2025;Suspensão;24/10/2025;23/10/2027;27/10/2025;Diário Oficial;;;No órgão sancionador;PREFEITURA MUNICIPAL DO RIO DE JANEIRO - RJ;RJ;MUNICIPAL;LEI 14133 - ART. 156;03/11/2025;Prefeitura;", ";")
}

func d(s string) time.Time {
	t, _ := time.Parse(time.DateOnly, s)
	return t
}

func TestParseCEISRow(t *testing.T) {
	s, err := ParseSanctionRow(columns(t, ceisHeader), ceisRow())

	if err != nil {
		t.Fatal(err)
	}
	if s.Register != RegisterCEIS || s.Code != "379001" || s.CNPJ != "28926250000176" || s.Name != "EMPRESA X COMERCIO LTDA" ||
		s.Category != "Suspensão" || !s.StartsAt.Equal(d("2025-10-24")) || !s.EndsAt.Equal(d("2027-10-23")) ||
		!s.PublishedAt.Equal(d("2025-10-27")) || s.Process != "SEI 123/2025" || s.Organ != "PREFEITURA MUNICIPAL DO RIO DE JANEIRO - RJ" ||
		s.OrganUF != "RJ" || s.Sphere != "MUNICIPAL" || s.Scope != "No órgão sancionador" || s.LegalBasis != "LEI 14133 - ART. 156" || s.FineCents != nil {
		t.Errorf("sanção: %+v", s)
	}
}

func TestParseCNEPRowWithFineAndNoEndDate(t *testing.T) {
	row := strings.Split("CNEP;312028;J;20773425000140;PALACIO X;;;;52262021;Multa;377157,39;18/04/2024;;18/04/2024;DOU;;;No órgão sancionador;Petrobras;RJ;FEDERAL;LEI 12846;18/07/2024;CGU;", ";")

	s, err := ParseSanctionRow(columns(t, cnepHeader), row)

	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "PALACIO X" || s.FineCents == nil || *s.FineCents != 37715739 || s.EndsAt != nil {
		t.Errorf("sanção: %+v", s)
	}
}

func TestPersonRowsAreNotCompanies(t *testing.T) {
	c := columns(t, ceisHeader)
	row := ceisRow()
	row[2], row[3] = "F", "77538650725"

	if c.IsCompany(row) {
		t.Error("pessoa física não é empresa")
	}
	if _, err := ParseSanctionRow(c, row); err == nil {
		t.Error("linha de pessoa física deveria ser recusada")
	}
}

func TestHeaderWithoutARequiredColumnIsRejected(t *testing.T) {
	if _, err := NewSanctionColumns(strings.Split(strings.Replace(ceisHeader, "DATA FINAL SANÇÃO", "FIM", 1), ";")); err == nil {
		t.Error("cabeçalho sem a data final deveria dar erro")
	}
}

func TestSanctionStates(t *testing.T) {
	end := d("2026-01-31")
	s := Sanction{EndsAt: &end, LastSeen: d("2026-09-25")}

	if got := s.State(d("2026-09-25"), d("2026-09-26")); got != SanctionEnded {
		t.Errorf("prazo passado: %s", got)
	}
	if got := s.State(d("2026-09-26"), d("2026-09-26")); got != SanctionDelisted {
		t.Errorf("fora do último arquivo: %s", got)
	}
	s.EndsAt = nil
	if got := s.State(d("2026-09-25"), d("2026-09-26")); got != SanctionListed {
		t.Errorf("sem data final, no cadastro: %s", got)
	}
}

func TestSanctionCoversDay(t *testing.T) {
	start, end := d("2025-01-10"), d("2025-12-31")
	s := Sanction{StartsAt: &start, EndsAt: &end}

	for day, want := range map[string]bool{"2025-01-09": false, "2025-01-10": true, "2025-12-31": true, "2026-01-01": false} {
		if got := s.CoversDay(d(day)); got != want {
			t.Errorf("%s: %v", day, got)
		}
	}
	s.EndsAt = nil
	if !s.CoversDay(d("2030-01-01")) {
		t.Error("sem data final cobre qualquer dia depois do início")
	}
}
