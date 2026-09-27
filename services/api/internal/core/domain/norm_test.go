package domain

import (
	"errors"
	"testing"
	"time"
)

const normPage = `<table><tr><td></td><td>Prefeitura Municipal de São Gonçalo</td></tr>
<tr><th>N.°</th><th>Categoria</th><th>Altera</th><th>Alterada por</th><th>Autor</th><th>Ementa</th><th>Promulgada</th><th>Anexos</th></tr>
<tr><td >1406/2022</td><td >Lei Ordinária </td><td >&nbsp;</td><td >&nbsp;</td><td >VEREADOR JALMIR JUNIOR PROJETO DE LEI 110/22</td>
<td >DISPOE SOBRE A CONSTITUIÇÃO DE  PATRIMÔNIO CULTURAL </td><td >08/12/2022</td>
<td ><button class='btn' onClick="MM_openBrWindow('formata_lei.php?Fnumero=1406/2022&Fpromulgacao=08/12/2022&categoria=01')" ></button></td></tr>
<tr><td >046/1994</td><td >Lei Ordinária</td><td></td><td></td><td></td><td>NÃO FOI EXPEDIDO.</td><td></td><td></td></tr>
<tr><td >1561/2025` + "\x1f" + ` b</td><td></td><td></td><td></td><td></td><td></td><td></td><td></td></tr>
<tr><td >245/202</td><td></td><td></td><td></td><td></td><td></td><td></td><td></td></tr></table>`

func TestParseNormsReadsNumberAuthorAndText(t *testing.T) {
	got, invalid, err := ParseNorms(NormLaw, []byte(normPage), "https://siapegov/leis/")

	if err != nil || len(got) != 3 || invalid != 1 {
		t.Fatalf("normas: %+v %v", got, err)
	}
	day := time.Date(2022, 12, 8, 0, 0, 0, 0, time.UTC)
	n := got[0]
	if n.Kind != NormLaw || n.Number != 1406 || n.Year != 2022 || n.Author != "VEREADOR JALMIR JUNIOR PROJETO DE LEI 110/22" ||
		n.Summary != "DISPOE SOBRE A CONSTITUIÇÃO DE PATRIMÔNIO CULTURAL" || n.PromulgatedOn == nil || !n.PromulgatedOn.Equal(day) ||
		n.TextURL != "https://siapegov/leis/formata_lei.php?Fnumero=1406/2022&Fpromulgacao=08/12/2022&categoria=01" {
		t.Errorf("lei: %+v", n)
	}
	if got[1].Number != 46 || got[1].PromulgatedOn != nil || got[1].TextURL != "" {
		t.Errorf("lei sem promulgação: %+v", got[1])
	}
	if got[2].Number != 1561 || got[2].Suffix != "B" || got[2].Label() != "1561/2025 B" {
		t.Errorf("sufixo e controle: %+v", got[2])
	}
}

func TestParseNormsRejectsAPageWithoutNorms(t *testing.T) {
	if _, _, err := ParseNorms(NormDecree, []byte(`<html>fora do ar</html>`), ""); err == nil {
		t.Error("página sem normas aceita")
	}
}

func TestParseNormNumberAndKind(t *testing.T) {
	for in, want := range map[string][2]int{"1.406/2022": {1406, 2022}, " 0497/2025": {497, 2025}, "110/22": {110, 2022}, "12/99": {12, 1999}} {
		n, y, _, err := ParseNormNumber(in)
		if err != nil || n != want[0] || y != want[1] {
			t.Errorf("%q: %d %d %v", in, n, y, err)
		}
	}
	for _, bad := range []string{"", "1406", "abc/2022", "0/2022"} {
		if _, _, _, err := ParseNormNumber(bad); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%q aceito", bad)
		}
	}
	if k, err := ParseNormKind(" Decreto "); err != nil || k != NormDecree {
		t.Errorf("decreto: %q %v", k, err)
	}
	if _, err := ParseNormKind("portaria"); !errors.Is(err, ErrInvalidInput) {
		t.Error("portaria aceita")
	}
}

func TestNormsWithoutRepeatsAndDiarioSearch(t *testing.T) {
	norms := []Norm{{Kind: NormLaw, Number: 46, Year: 1994, Summary: "a"}, {Kind: NormDecree, Number: 46, Year: 1994}, {Kind: NormLaw, Number: 46, Year: 1994},
		{Kind: NormLaw, Number: 46, Year: 1994, Suffix: "A"}}

	got, repeated := WithoutRepeatedNorms(norms)

	if repeated != 1 || len(got) != 3 || got[0].Summary != "a" {
		t.Errorf("repetidas: %+v %d", got, repeated)
	}
	if s := NormDiarioSearch(Norm{Number: 1406, Year: 2022}); s != `"1.406/2022"` {
		t.Errorf("busca: %s", s)
	}
	if s := NormDiarioSearch(Norm{Number: 497, Year: 2025}); s != `"497/2025"` {
		t.Errorf("busca: %s", s)
	}
}
